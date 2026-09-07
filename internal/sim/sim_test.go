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
