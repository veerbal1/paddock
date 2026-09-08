// Package telemetry holds the wire contract between the device fleet and the
// backend. Both sides import it; neither imports the other.
package telemetry

import (
	"encoding/json"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

type Activity int

const (
	Grazing Activity = iota
	Walking
	Resting
)

func (a Activity) String() string {
	switch a {
	case Grazing:
		return "grazing"
	case Walking:
		return "walking"
	case Resting:
		return "resting"
	default:
		return "unknown"
	}
}

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
	CowID    string
	Pos      geom.Point
	At       time.Time
	State    State
	Cue      Cue
	Activity Activity
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

// MarshalJSON sends the state over the wire as a name rather than a number.
// encoding/json ignores String() — it looks for this method instead — so
// without it a ping would carry "State": 2 and every reader would need a
// lookup table.
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON reads the name back into a number. Marshal without this
// is a one-way bridge: publish works, subscribe silently drops.
func (s *State) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return err
	}
	switch name {
	case "inside":
		*s = Inside
	case "warning":
		*s = Warning
	case "breached":
		*s = Breached
	case "escaped":
		*s = Escaped
	default:
		*s = Inside
	}
	return nil
}

// MarshalJSON sends the cue over the wire as a name, for the same reason.
func (c Cue) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// UnmarshalJSON reads the cue name back.
func (c *Cue) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return err
	}
	switch name {
	case "none":
		*c = CueNone
	case "audio":
		*c = CueAudio
	case "vibration":
		*c = CueVibration
	case "pulse":
		*c = CuePulse
	default:
		*c = CueNone
	}
	return nil
}

// MarshalJSON sends the activity over the wire as a name, for the same reason.
func (a Activity) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON reads the activity name back.
func (a *Activity) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return err
	}
	switch name {
	case "grazing":
		*a = Grazing
	case "walking":
		*a = Walking
	case "resting":
		*a = Resting
	default:
		*a = Grazing
	}
	return nil
}
