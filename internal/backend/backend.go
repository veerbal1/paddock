// Package backend is the cloud side. It consumes telemetry from the device
// fleet and never reaches back into it.
package backend

import (
	"fmt"

	"github.com/veerbal1/paddock/internal/telemetry"
)

type Backend struct {
	latest map[string]telemetry.Ping
}

func New() *Backend {
	return &Backend{
		latest: make(map[string]telemetry.Ping),
	}
}

// Consume reads pings until the channel is closed and drained.
func (b *Backend) Consume(pings <-chan telemetry.Ping) {
	for p := range pings {
		fmt.Printf("%s  x=%.1f y=%.1f %s\n", p.CowID, p.Pos.X, p.Pos.Y, p.State)
	}
}
