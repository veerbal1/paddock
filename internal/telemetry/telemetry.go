// Package telemetry holds the wire contract between the device fleet and the
// backend. Both sides import it; neither imports the other.
package telemetry

import (
	"time"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

// Ping is one position report leaving a collar. Only what genuinely crosses
// the device boundary belongs here — no heading, no seed, nothing internal to
// the simulation.
type Ping struct {
	CowID string
	Pos   geom.Point
	At    time.Time
	State fence.State
}
