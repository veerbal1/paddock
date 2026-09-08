package sim

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
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

// stubClock is the fake Clock from Step 7: every ping gets the same
// timestamp, so two runs can be compared ping-for-ping.
type stubClock struct{ t time.Time }

func (c stubClock) Now() time.Time { return c.t }

func collectPings(seed int64, cows, ticks int) []telemetry.Ping {
	f := fence.Rect{MinX: 0, MaxX: 500, MinY: 0, MaxY: 500}
	s := New(cows, seed, geom.Point{X: 250, Y: 250}, f)
	s.Speed = 60
	s.Clock = stubClock{t: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go s.Run(ctx)

	// Wall-clock tick boundaries are NOT deterministic (a tick landing
	// exactly on the timeout is a select coin-flip), so take a fixed prefix:
	// the first ticks*cows pings are fully determined by the seed.
	want := ticks * cows
	var got []telemetry.Ping
	for p := range s.Pings() {
		got = append(got, p)
		if len(got) == want {
			cancel()
		}
	}
	if len(got) < want {
		return got
	}
	return got[:want]
}

// TestSeedReproducesBreachSequence is the Step 7 proof: --speed=60 --seed=42
// twice must yield the same breach sequence. Speed changes only how many
// physics substeps run per tick; the RNG stream is untouched.
func TestSeedReproducesBreachSequence(t *testing.T) {
	a := collectPings(42, 10, 2)
	b := collectPings(42, 10, 2)
	if len(a) != 20 || len(b) != 20 {
		t.Fatalf("collected %d vs %d pings, want 20 each", len(a), len(b))
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed gave different ping sequences")
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
