package sim

import (
	"math"
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

// herdSpread returns the farthest any cow is from the herd centroid, in metres.
func herdSpread(s *Sim) float64 {
	c := s.Centroid()
	max := 0.0
	for _, u := range s.units {
		dx := u.cow.Pos.X - c.X
		dy := u.cow.Pos.Y - c.Y
		if d := math.Hypot(dx, dy); d > max {
			max = d
		}
	}
	return max
}

// TestHerdStaysTogether runs 1000 ticks sequentially and checks the herd
// hasn't scattered. Sequential on purpose: no goroutines, no race, just
// the cohesion math.
func TestHerdStaysTogether(t *testing.T) {
	f := fence.Rect{MinX: 0, MaxX: 500, MinY: 0, MaxY: 500}
	s := New(50, 42, geom.Point{X: 250, Y: 250}, f)

	for i := 0; i < 1000; i++ {
		c := s.Centroid()
		for _, u := range s.units {
			u.cow.Step(time.Second, c)
		}
	}

	if got := herdSpread(s); got > 200 {
		t.Fatalf("herd spread = %.1fm, want <= 200m", got)
	}
}

// TestSetFenceUpdatesEveryCollar checks the PUT /fence downlink: one call
// must replace all 50 collar copies plus the getter, so the next tick
// enforces the new bounds everywhere, not halfway through the herd.
func TestSetFenceUpdatesEveryCollar(t *testing.T) {
	old := fence.Rect{MinX: 0, MaxX: 500, MinY: 0, MaxY: 500}
	s := New(50, 42, geom.Point{X: 250, Y: 250}, old)

	next := fence.Rect{MinX: 200, MaxX: 300, MinY: 200, MaxY: 300}
	s.SetFence(next)

	if got := s.Fence(); got != next {
		t.Fatalf("Fence() = %+v, want %+v", got, next)
	}
	for _, u := range s.units {
		if u.collar.Fence != next {
			t.Fatalf("collar %s still has %+v, want %+v", u.cow.ID, u.collar.Fence, next)
		}
	}
}
