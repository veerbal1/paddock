package collar

import (
	"github.com/veerbal1/paddock/internal/shared/fence"
	"github.com/veerbal1/paddock/internal/shared/geom"
	"github.com/veerbal1/paddock/internal/shared/telemetry"
)

// Collar is one device's state machine. It is not safe for concurrent use:
// whoever owns the collar (the sim today, the device loop tomorrow) is
// responsible for serialising Observe and SetFence.
type Collar struct {
	CowID  string
	bounds fence.Rect
	state  telemetry.State
	warnM  float64

	// cue state
	dwell  int
	level  telemetry.Cue
	gaveUp bool
}

const escalateAfter = 8

func New(cowID string, f fence.Rect, warnM float64) *Collar {
	return &Collar{CowID: cowID, bounds: f, state: telemetry.Inside, warnM: warnM}
}

// SetFence replaces the bounds this collar enforces. It takes effect on the
// next Observe. Loop 3's retained downlink lands here; until then the sim
// calls it. Keeping the field private means no caller can swap the fence
// half-way through a tick's evaluation.
func (c *Collar) SetFence(f fence.Rect) { c.bounds = f }

// Fence returns the bounds currently enforced. The collar's own copy is the
// truth on the device — the cloud's copy is only what was authored.
func (c *Collar) Fence() fence.Rect { return c.bounds }

// Observation is the full record of one tick: what the collar believed before,
// what it believes now, and what it did about it.
type Observation struct {
	From    telemetry.State
	To      telemetry.State
	Changed bool
	Cue     telemetry.Cue
}

func (c *Collar) Observe(p geom.Point) Observation {
	prev := c.state
	zone := c.bounds.Evaluate(p, c.warnM)
	var cue telemetry.Cue

	var newState telemetry.State
	switch zone {
	case fence.ZoneInside:
		newState = telemetry.Inside
	case fence.ZoneWarning:
		newState = telemetry.Warning
	case fence.ZoneOutside:
		newState = telemetry.Breached
	}

	// The comparisons below lean on telemetry.State being ordered
	// inside < warning < breached < escaped: ">" means "further out".
	if newState > prev {
		// Heading further out. Only start the ladder if it is not already
		// climbing — pushing deeper must not reset it back to audio.
		if c.level == telemetry.CueNone {
			c.dwell = 0
			c.level = telemetry.CueAudio
			cue = telemetry.CueAudio
		}
	} else if newState < prev {
		// cow coming inside, do nothing, reset cues
		c.dwell = 0
		c.level = telemetry.CueNone
		c.gaveUp = false
		cue = telemetry.CueNone
	} else {
		c.dwell++
		if c.level != telemetry.CueNone && !c.gaveUp && c.dwell%escalateAfter == 0 {
			level := c.level + 1
			c.level = level
			if level > telemetry.CuePulse {
				c.gaveUp = true
				c.level = telemetry.CueNone
				cue = telemetry.CueNone
			} else {
				cue = level
			}
		}
	}
	c.state = newState

	return Observation{
		From:    prev,
		To:      newState,
		Changed: newState != prev,
		Cue:     cue,
	}
}
