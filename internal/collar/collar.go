package collar

import (
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

type Collar struct {
	CowID string
	Fence fence.Rect
	state fence.State
}

func New(cowID string, f fence.Rect) *Collar {
	return &Collar{CowID: cowID, Fence: f, state: fence.Inside}
}

func (c *Collar) Observe(p geom.Point) (fence.State, bool) {
	var newState fence.State
	if c.Fence.Contains(p) {
		newState = fence.Inside
	} else {
		newState = fence.Breached
	}

	changed := newState != c.state
	if changed {
		c.state = newState
	}
	return newState, changed
}
