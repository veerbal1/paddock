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
}

func New(cowID string, f fence.Rect, warnM float64) *Collar {
	return &Collar{CowID: cowID, Fence: f, state: telemetry.Inside, warnM: warnM}
}

func (c *Collar) Observe(p geom.Point) (telemetry.State, bool) {
	zone := c.Fence.Evaluate(p, c.warnM)

	var newState telemetry.State
	switch zone {
	case fence.ZoneInside:
		newState = telemetry.Inside
	case fence.ZoneWarning:
		newState = telemetry.Warning
	case fence.ZoneOutside:
		newState = telemetry.Breached
	}

	changed := newState != c.state
	c.state = newState

	return newState, changed
}
