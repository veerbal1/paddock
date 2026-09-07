// Package backend is the cloud side. It consumes telemetry from the device
// fleet and never reaches back into it.
package backend

import (
	"fmt"
	"sync"

	"github.com/veerbal1/paddock/internal/telemetry"
)

type Backend struct {
	latest map[string]telemetry.Ping
	mu     sync.RWMutex
}

func New() *Backend {
	return &Backend{
		latest: make(map[string]telemetry.Ping),
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

// Consume reads pings until the channel is closed and drained.
func (b *Backend) Consume(pings <-chan telemetry.Ping) {
	for p := range pings {
		b.mu.Lock()
		b.latest[p.CowID] = p
		b.mu.Unlock()
		fmt.Printf("%s  x=%.1f y=%.1f  %-8s  cue=%s\n",
			p.CowID, p.Pos.X, p.Pos.Y, p.State, p.Cue)
	}
}
