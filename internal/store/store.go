package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/veerbal1/paddock/internal/telemetry"
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
func (s *Store) SavePing(ctx context.Context, farmID string, p telemetry.Ping) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO latest (farm_id, cow_id, x, y, state, cue, activity, at, fence_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (farm_id, cow_id) DO UPDATE SET
			x=EXCLUDED.x, y=EXCLUDED.y, state=EXCLUDED.state,
			cue=EXCLUDED.cue, activity=EXCLUDED.activity, at=EXCLUDED.at,
			fence_version=EXCLUDED.fence_version, updated_at=now()
	`, farmID, p.CowID, p.Pos.X, p.Pos.Y,
		p.State.String(), p.Cue.String(), p.Activity.String(), p.At, 1)
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
