// Package backend is the cloud side. It consumes telemetry from the device
// fleet and never reaches back into it.
package backend

import (
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

// maxAlerts bounds memory. Retention policy is Loop 4's problem; this just
// stops an unbounded slice. Open episodes are never dropped.
const maxAlerts = 200

type Backend struct {
	latest map[string]telemetry.Ping
	alerts []Alert
	open   map[string]int // cow -> index into alerts while out
	mu     sync.RWMutex
}

func New() *Backend {
	return &Backend{
		latest: make(map[string]telemetry.Ping),
		open:   make(map[string]int),
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

// Consume reads pings until the channel is closed and drained.
func (b *Backend) Consume(pings <-chan telemetry.Ping) {
	for p := range pings {
		b.mu.Lock()
		prev, seen := b.latest[p.CowID]
		b.latest[p.CowID] = p

		wasOut := seen && prev.State != telemetry.Inside
		isOut := p.State != telemetry.Inside
		switch {
		case !wasOut && isOut:
			// Left inside (or first ever seen out): open an episode.
			b.open[p.CowID] = len(b.alerts)
			b.alerts = append(b.alerts, Alert{CowID: p.CowID, State: p.State, StartedAt: p.At})
			b.prune()
		case wasOut && !isOut:
			// Back inside: close it.
			a := &b.alerts[b.open[p.CowID]]
			t := p.At
			a.EndedAt = &t
			a.State = p.State
			delete(b.open, p.CowID)
		case wasOut && isOut:
			// Still out: track the latest state.
			b.alerts[b.open[p.CowID]].State = p.State
		}
		b.mu.Unlock()
	}
}

// prune drops the oldest closed episodes past maxAlerts. Caller holds mu.
func (b *Backend) prune() {
	for len(b.alerts) > maxAlerts && b.alerts[0].EndedAt != nil {
		b.alerts = b.alerts[1:]
		for cow, i := range b.open {
			b.open[cow] = i - 1
		}
	}
}
