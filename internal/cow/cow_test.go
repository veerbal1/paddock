package cow

import (
	"math"
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

func TestCowIsDeterministic(t *testing.T) {
	start := geom.Point{X: 100, Y: 100}

	a := New("cow-01", start, 42, 1.0)
	b := New("cow-01", start, 42, 1.0)

	for i := 0; i < 100; i++ {
		a.Step(time.Second, geom.Point{X: 50, Y: 50})
		b.Step(time.Second, geom.Point{X: 50, Y: 50})

		if a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y {
			t.Fatalf("step %d: diverged: a=%v b=%v", i, a.Pos, b.Pos)
		}
	}
}

func TestCowDifferentSeedsDiverge(t *testing.T) {
	start := geom.Point{X: 100, Y: 100}

	a := New("cow-01", start, 42, 1.0)
	b := New("cow-02", start, 43, 1.0)

	for i := 0; i < 100; i++ {
		a.Step(time.Second, geom.Point{X: 50, Y: 50})
		b.Step(time.Second, geom.Point{X: 50, Y: 50})

		if a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y {
			return
		}
	}

	t.Fatal("paths never diverged after 100 steps")
}

// angleBetween returns the shortest angle between two headings, in radians.
// Without this a turn from 6.1 to 0.2 looks like 5.9 radians instead of 0.38.
func angleBetween(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > math.Pi {
		d = 2*math.Pi - d
	}
	return d
}

// TestCowTurnsAwayWhenCued checks that a fully trained cow given a certain cue
// reverses. It must turn roughly 180°, not to some absolute direction: the cow
// has no idea where the fence is, it only knows to go back the way it came.
func TestCowTurnsAwayWhenCued(t *testing.T) {
	c := New("cow-01", geom.Point{X: 100, Y: 100}, 42, 1.0)

	for i := 0; i < 50; i++ {
		before := c.Heading

		if !c.TurnAway(1.0) {
			t.Fatalf("turn %d: fully trained cow ignored a certain cue", i)
		}

		if got := angleBetween(before, c.Heading); got < math.Pi-turnSpread {
			t.Errorf("turn %d: turned %.2f rad, want at least %.2f",
				i, got, math.Pi-turnSpread)
		}
	}
}

// TestCowIgnoresCueWhenUntrained is the other half: an untrained cow does not
// respond, and does not move its heading either.
func TestCowIgnoresCueWhenUntrained(t *testing.T) {
	c := New("cow-01", geom.Point{X: 100, Y: 100}, 42, 0.0)

	for i := 0; i < 50; i++ {
		before := c.Heading

		if c.TurnAway(1.0) {
			t.Fatalf("turn %d: untrained cow complied", i)
		}
		if c.Heading != before {
			t.Fatalf("turn %d: heading moved on an ignored cue: %v -> %v",
				i, before, c.Heading)
		}
	}
}

// TestCowComplianceFollowsProbability checks the middle ground: over many cues
// a half-trained cow should comply roughly half the time. This is the property
// the cue compliance metric will be built on.
func TestCowComplianceFollowsProbability(t *testing.T) {
	c := New("cow-01", geom.Point{X: 100, Y: 100}, 42, 0.5)

	const trials = 10000
	complied := 0
	for i := 0; i < trials; i++ {
		if c.TurnAway(1.0) {
			complied++
		}
	}

	rate := float64(complied) / trials
	if rate < 0.45 || rate > 0.55 {
		t.Errorf("compliance rate = %.3f, want about 0.5", rate)
	}
}

// TestCowHeadingStaysWrapped guards the invariant that keeps logs readable and
// float precision from drifting over a long run.
func TestCowHeadingStaysWrapped(t *testing.T) {
	c := New("cow-01", geom.Point{X: 100, Y: 100}, 42, 1.0)

	for i := 0; i < 1000; i++ {
		c.Step(time.Second, geom.Point{X: 50, Y: 50})
		c.TurnAway(0.5)

		if c.Heading < 0 || c.Heading >= 2*math.Pi {
			t.Fatalf("step %d: heading out of range: %v", i, c.Heading)
		}
	}
}
