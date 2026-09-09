package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/veerbal1/paddock/internal/shared/telemetry"
)

// Store is the disk behind the backend. Handlers talk to this,
// never to pgx directly — SQL lives in one package, tested once.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a pool of connections to Postgres. dsn looks like
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

// Close drains the pool.
func (s *Store) Close() {
	s.pool.Close()
}

// SavePing overwrites the cow's latest row. Table never grows past
// one row per cow: INSERT ... ON CONFLICT DO UPDATE.
//
// fence_version is deliberately not written: no collar reports one yet, so
// the column keeps its schema default until Step 3 puts it on the wire.
// Writing a hardcoded 1 here would be a column that lies.
func (s *Store) SavePing(ctx context.Context, farmID string, p telemetry.Ping) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO latest (farm_id, cow_id, x, y, state, cue, activity, at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (farm_id, cow_id) DO UPDATE SET
			x=EXCLUDED.x, y=EXCLUDED.y, state=EXCLUDED.state,
			cue=EXCLUDED.cue, activity=EXCLUDED.activity, at=EXCLUDED.at,
			updated_at=now()
	`, farmID, p.CowID, p.Pos.X, p.Pos.Y,
		p.State.String(), p.Cue.String(), p.Activity.String(), p.At)
	return err
}

// Latest returns every cow's last ping for one farm. Never leaks
// another farm's rows: the WHERE is the wall.
func (s *Store) Latest(ctx context.Context, farmID string) (map[string]telemetry.Ping, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cow_id, x, y, state, cue, activity, at
		FROM latest WHERE farm_id=$1
	`, farmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]telemetry.Ping{}
	for rows.Next() {
		var p telemetry.Ping
		var state, cue, activity string
		if err := rows.Scan(&p.CowID, &p.Pos.X, &p.Pos.Y, &state, &cue, &activity, &p.At); err != nil {
			return nil, err
		}
		p.State = parseState(state)
		p.Cue = parseCue(cue)
		p.Activity = parseActivity(activity)
		out[p.CowID] = p
	}
	return out, rows.Err()
}

// parseState turns the stored name back into a number. Same table as
// telemetry.State.UnmarshalJSON — kept local so store never imports
// JSON concerns for a SQL read.
func parseState(s string) telemetry.State {
	switch s {
	case "inside":
		return telemetry.Inside
	case "warning":
		return telemetry.Warning
	case "breached":
		return telemetry.Breached
	case "escaped":
		return telemetry.Escaped
	}
	return telemetry.Inside
}

func parseCue(s string) telemetry.Cue {
	switch s {
	case "audio":
		return telemetry.CueAudio
	case "vibration":
		return telemetry.CueVibration
	case "pulse":
		return telemetry.CuePulse
	}
	return telemetry.CueNone
}

func parseActivity(s string) telemetry.Activity {
	switch s {
	case "grazing":
		return telemetry.Grazing
	case "walking":
		return telemetry.Walking
	case "resting":
		return telemetry.Resting
	}
	return telemetry.Grazing
}

// OpenAlert writes a new breach episode and returns its id. If the cow
// already has an open episode, the unique index refuses and we return
// the existing id instead of erroring — two backends racing must not
// create two cases for one escape.
func (s *Store) OpenAlert(ctx context.Context, farmID, cowID string, state telemetry.State, startedAt time.Time) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO alerts (farm_id, cow_id, state, started_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (farm_id, cow_id) WHERE ended_at IS NULL
		DO UPDATE SET state=EXCLUDED.state
		RETURNING id
	`, farmID, cowID, state.String(), startedAt).Scan(&id)
	return id, err
}

// CloseAlert stamps the open episode shut. Returns true if it closed
// one, false if nothing was open (already closed, or never opened).
func (s *Store) CloseAlert(ctx context.Context, farmID, cowID string, endedAt time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE alerts SET ended_at=$3, state='inside'
		WHERE farm_id=$1 AND cow_id=$2 AND ended_at IS NULL
	`, farmID, cowID, endedAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Alert is one breach episode as it sits on disk. The backend keeps its own
// domain type with the same shape; this is only the row. Duplicating four
// fields is the price of the arrow pointing one way — store must never
// import backend.
type Alert struct {
	CowID     string
	State     telemetry.State
	StartedAt time.Time
	EndedAt   *time.Time
}

// ListAlerts returns a farm's most recent episodes, oldest first — the order
// the backend replays them in at boot. limit bounds the read so a farm with a
// year of history cannot blow up a restart.
func (s *Store) ListAlerts(ctx context.Context, farmID string, limit int) ([]Alert, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cow_id, state, started_at, ended_at FROM (
			SELECT id, cow_id, state, started_at, ended_at
			FROM alerts WHERE farm_id=$1
			ORDER BY started_at DESC, id DESC
			LIMIT $2
		) recent
		ORDER BY started_at ASC, id ASC
	`, farmID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Alert, 0, limit)
	for rows.Next() {
		var a Alert
		var state string
		if err := rows.Scan(&a.CowID, &state, &a.StartedAt, &a.EndedAt); err != nil {
			return nil, err
		}
		a.State = parseState(state)
		out = append(out, a)
	}
	return out, rows.Err()
}

// SaveFence writes a new fence version and returns the version number.
// Two steps in one transaction: take the paddock's lock, then insert max+1.
// The lock means two PUTs racing wait in line instead of colliding.
//
// The lock is an advisory one, not SELECT ... FOR UPDATE: row locks only
// exist for rows that are already there, so they cannot stop two writers
// both inserting the very first version. An advisory lock is taken on the
// paddock's name, which exists before any row does. (The earlier
// `SELECT MAX(...) FOR UPDATE` was not merely weak, it was rejected outright
// by Postgres — FOR UPDATE is not allowed with aggregates.)
func (s *Store) SaveFence(ctx context.Context, farmID, paddockID string, polygon []byte) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1 || '/' || $2)::bigint)`,
		farmID, paddockID); err != nil {
		return 0, err
	}

	var max int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version),0) FROM fences
		WHERE farm_id=$1 AND paddock_id=$2
	`, farmID, paddockID).Scan(&max)
	if err != nil {
		return 0, err
	}

	next := max + 1
	_, err = tx.Exec(ctx, `
		INSERT INTO fences (farm_id, paddock_id, version, polygon)
		VALUES ($1,$2,$3,$4)
	`, farmID, paddockID, next, polygon)
	if err != nil {
		return 0, err
	}
	return next, tx.Commit(ctx)
}

// LatestFence returns the newest version and its polygon. False when
// the paddock was never drawn.
func (s *Store) LatestFence(ctx context.Context, farmID, paddockID string) (int, []byte, bool, error) {
	var v int
	var poly []byte
	err := s.pool.QueryRow(ctx, `
		SELECT version, polygon FROM fences
		WHERE farm_id=$1 AND paddock_id=$2
		ORDER BY version DESC LIMIT 1
	`, farmID, paddockID).Scan(&v, &poly)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	return v, poly, true, nil
}
