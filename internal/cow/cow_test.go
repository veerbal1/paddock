package cow

import (
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

func TestCowIsDeterministic(t *testing.T) {
	start := geom.Point{X: 100, Y: 100}

	a := New("cow-01", start, 42)
	b := New("cow-01", start, 42)

	for i := 0; i < 100; i++ {
		a.Step(time.Second)
		b.Step(time.Second)

		if a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y {
			t.Fatalf("step %d: diverged: a=%v b=%v", i, a.Pos, b.Pos)
		}
	}
}

func TestCowDifferentSeedsDiverge(t *testing.T) {
	start := geom.Point{X: 100, Y: 100}

	a := New("cow-01", start, 42)
	b := New("cow-02", start, 43)

	for i := 0; i < 100; i++ {
		a.Step(time.Second)
		b.Step(time.Second)

		if a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y {
			return
		}
	}

	t.Fatal("paths never diverged after 100 steps")
}
