// Package telemetry holds the wire contract between the device fleet and the
// backend. Both sides import it; neither imports the other.
package telemetry

import (
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

// State is what a collar believes about its cow. Unlike a fence.Zone, this has
// memory: hysteresis and dwell time hold it steady while the raw zone flickers.
type State int

const (
	Inside State = iota
	Warning
	Breached
	Escaped
)

func (s State) String() string {
	switch s {
	case Inside:
		return "inside"
	case Warning:
		return "warning"
	case Breached:
		return "breached"
	case Escaped:
		return "escaped"
	default:
		return "unknown"
	}
}

// Ping is one position report leaving a collar. Only what genuinely crosses
// the device boundary belongs here — no heading, no seed, nothing internal to
// the simulation.
type Ping struct {
	CowID string
	Pos   geom.Point
	At    time.Time
	State State
	Cue   Cue
}

// Cue is what the collar did to the animal on this tick. The ladder only
// climbs while the cow keeps heading outward; coming back in resets it.
type Cue int

const (
	CueNone Cue = iota
	CueAudio
	CueVibration
	CuePulse
)

func (c Cue) String() string {
	switch c {
	case CueNone:
		return "none"
	case CueAudio:
		return "audio"
	case CueVibration:
		return "vibration"
	case CuePulse:
		return "pulse"
	default:
		return "unknown"
	}
}
