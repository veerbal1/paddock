package collar

import (
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

type Collar struct {
	CowID string
	Fence fence.Rect
	state telemetry.State
	warnM float64

	// cue state
	dwell  int
	level  telemetry.Cue
	gaveUp bool
}

const escalateAfter = 8

func New(cowID string, f fence.Rect, warnM float64) *Collar {
	return &Collar{CowID: cowID, Fence: f, state: telemetry.Inside, warnM: warnM}
}

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
	zone := c.Fence.Evaluate(p, c.warnM)
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

	if newState > prev {
		// cow going outside
		if c.level == telemetry.CueNone {
			c.dwell = 0
			c.level = telemetry.CueAudio
			cue = telemetry.CueAudio
		} else {
			// Do nothing here.
			// Mean it is already in Cue state, then do nothing. keep it running, dwell keep increasing.
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
