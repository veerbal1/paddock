// Package backend is the cloud side. It consumes telemetry from the device
// fleet and never reaches back into it.
package backend

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/veerbal1/paddock/internal/telemetry"
)

// Alert is one breach episode: a cow left `inside` at StartedAt and came
// back at EndedAt. EndedAt is nil while she is still out.
type Alert struct {
	CowID     string
	State     telemetry.State // latest state seen in this episode
	StartedAt time.Time
	EndedAt   *time.Time
}

// Open reports whether the cow is still out.
func (a Alert) Open() bool { return a.EndedAt == nil }

// MaxAlerts bounds memory. Retention policy is Loop 4's problem; this just
// stops an unbounded slice. Open episodes are never dropped, so a herd stuck
// outside can still push past this — see prune.
const MaxAlerts = 200

// Store is the disk behind the backend, declared here because this is the
// package that uses it. *store.Store satisfies it; so does a fake in a test,
// which is the whole point — the episode logic is provable without Postgres.
//
// Reads are absent on purpose: history comes back through Hydrate at boot,
// so nothing in the hot path ever waits on a SELECT.
type Store interface {
	SavePing(ctx context.Context, farmID string, p telemetry.Ping) error
	OpenAlert(ctx context.Context, farmID, cowID string, state telemetry.State, startedAt time.Time) (int64, error)
	CloseAlert(ctx context.Context, farmID, cowID string, endedAt time.Time) (bool, error)
}

type Backend struct {
	latest map[string]telemetry.Ping
	alerts []Alert
	open   map[string]int // cow -> index into alerts while out
	mu     sync.RWMutex

	st Store // nil in unit tests: memory only, no disk
}

func New() *Backend {
	return &Backend{
		latest: make(map[string]telemetry.Ping),
		open:   make(map[string]int),
	}
}

// WithStore plugs disk behind memory. Chain it: backend.New().WithStore(st).
// Farm comes from each ping's topic stamp, never from config.
func (b *Backend) WithStore(st Store) *Backend {
	b.st = st
	return b
}

// Hydrate replays episodes read from disk back into memory. Call it once at
// boot, before Consume. GET /alerts answers from memory, so without this a
// restart looks like a farm that never had an escape.
func (b *Backend) Hydrate(alerts []Alert) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.alerts = make([]Alert, len(alerts))
	copy(b.alerts, alerts)
	b.open = make(map[string]int, len(alerts))
	for i, a := range b.alerts {
		if a.Open() {
			// A cow still out when we died is still out now. Keeping the
			// index means her return closes this episode instead of
			// opening a second one.
			b.open[a.CowID] = i
		}
	}
}

func (b *Backend) Latest() map[string]telemetry.Ping {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make(map[string]telemetry.Ping, len(b.latest))
	for k, val := range b.latest {
		out[k] = val
	}
	return out
}

// Alerts returns every breach episode, oldest first, still-out first-seen
// order preserved. EndedAt is nil on episodes that are still open.
func (b *Backend) Alerts() []Alert {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]Alert, len(b.alerts))
	for i, a := range b.alerts {
		out[i] = a
		if a.EndedAt != nil {
			t := *a.EndedAt
			out[i].EndedAt = &t
		}
	}
	return out
}

// Consume reads pings until the channel closes or ctx is cancelled. On
// cancel it drains what the broker already handed over, then returns: a
// ping that made it into the buffer is one we accepted, so dropping it on
// shutdown would be a lost report we can never explain.
func (b *Backend) Consume(ctx context.Context, pings <-chan telemetry.Ping) {
	for {
		select {
		case p, ok := <-pings:
			if !ok {
				return
			}
			b.handle(p)
		case <-ctx.Done():
			for {
				select {
				case p, ok := <-pings:
					if !ok {
						return
					}
					b.handle(p)
				default:
					return
				}
			}
		}
	}
}

// handle is one ping's journey: memory first (fast, under the lock), disk
// second (slow, outside it). Splitting the two is what keeps a stalled
// database from freezing the next cow's episode logic.
func (b *Backend) handle(p telemetry.Ping) {
	opened, closed := b.record(p)
	b.persist(p, opened, closed)
}

// record is the episode state machine, and nothing else. Pure memory, so
// every rule in here is testable without a database.
func (b *Backend) record(p telemetry.Ping) (opened, closed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.latest[p.CowID] = p

	// "Was she out?" is answered by the open-episode map, not by her last
	// position. The two agree while the process runs, but only the map
	// survives a restart — and after Hydrate, a cow who was out when we
	// died must have her return close that episode instead of opening a
	// second one.
	_, wasOut := b.open[p.CowID]
	isOut := p.State != telemetry.Inside

	switch {
	case !wasOut && isOut:
		// Left inside (or first ever seen out): open an episode.
		b.open[p.CowID] = len(b.alerts)
		b.alerts = append(b.alerts, Alert{CowID: p.CowID, State: p.State, StartedAt: p.At})
		b.prune()
		opened = true
	case wasOut && !isOut:
		// Back inside: close it.
		a := &b.alerts[b.open[p.CowID]]
		t := p.At
		a.EndedAt = &t
		a.State = p.State
		delete(b.open, p.CowID)
		closed = true
	case wasOut && isOut:
		// Still out: track the latest state.
		b.alerts[b.open[p.CowID]].State = p.State
	}
	return opened, closed
}

// persist mirrors the decision onto disk. Failures are logged, not fatal —
// but note the honest cost: memory and disk can drift apart here, and
// memory is what GET /alerts answers from. Reconciling the two is Loop 4's
// job, once the outbox exists.
func (b *Backend) persist(p telemetry.Ping, opened, closed bool) {
	if b.st == nil {
		return
	}

	// Background, not the caller's context: a ping accepted before shutdown
	// still deserves to reach disk.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := b.st.SavePing(ctx, p.FarmID, p); err != nil {
		log.Printf("store: save %s: %v", p.CowID, err)
	}
	if opened {
		if _, err := b.st.OpenAlert(ctx, p.FarmID, p.CowID, p.State, p.At); err != nil {
			log.Printf("store: open %s: %v", p.CowID, err)
		}
	}
	if closed {
		if _, err := b.st.CloseAlert(ctx, p.FarmID, p.CowID, p.At); err != nil {
			log.Printf("store: close %s: %v", p.CowID, err)
		}
	}
}

// prune drops the oldest closed episodes past MaxAlerts. Caller holds mu.
// It stops at the first open episode: a cow still out is never forgotten.
// The bound therefore holds only while the oldest episodes keep closing —
// a permanently open head means the slice keeps growing, which is a real
// (accepted) hole until retention lands in Loop 4.
func (b *Backend) prune() {
	for len(b.alerts) > MaxAlerts && b.alerts[0].EndedAt != nil {
		b.alerts = b.alerts[1:]
		for cow, i := range b.open {
			b.open[cow] = i - 1
		}
	}
}
