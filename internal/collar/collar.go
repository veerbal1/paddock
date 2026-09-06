package collar

import (
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

type Collar struct {
	CowID string
	Fence fence.Rect
	state fence.State
	warnM float64
}

func New(cowID string, f fence.Rect, warnM float64) *Collar {
	return &Collar{CowID: cowID, Fence: f, state: fence.Inside, warnM: warnM}
}

func (c *Collar) Observe(p geom.Point) (fence.State, bool) {
	newState := c.Fence.Evaluate(p, c.warnM)

	changed := newState != c.state

	if changed {
		c.state = newState
	}

	return newState, changed
}
