package cow

import (
	"math"
	"math/rand"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

const (
	// turnRate is how far the cow can swing its heading in one second, in radians.
	turnRate = 0.6

	// defaultSpeedMS is a grazing cow: slow, and mostly not going anywhere.
	defaultSpeedMS = 0.3

	// turnSpread is how far a cued cow's about-turn strays from a clean
	// reversal, in radians. A spooked animal does not pivot exactly 180°.
	turnSpread = 0.5
)

// Cow is a simulated animal. Pos is its true position - only the
// simulator knows this. A collar reads it through GPS, with error.
type Cow struct {
	ID      string
	Pos     geom.Point
	rng     *rand.Rand
	Heading float64 // in radian, cow's face direction
	SpeedMS float64
	Trained float64 // 0..1, how reliably this cow obeys a cue
}

// New returns a cow at start with its own seeded random source, so the
// same seed always produces the same path.
func New(id string, start geom.Point, seed int64, trained float64) *Cow {
	rng := rand.New(rand.NewSource(seed))
	return &Cow{
		ID:      id,
		Pos:     start,
		Heading: rng.Float64() * 2 * math.Pi,
		SpeedMS: defaultSpeedMS,
		Trained: trained,
		rng:     rng,
	}
}

// Step advances the cow by one tick of simulated time dt.
func (c *Cow) Step(dt time.Duration, center geom.Point) {
	dx := center.X - c.Pos.X
	dy := center.Y - c.Pos.Y

	compass := math.Atan2(dx, dy)
	c.Heading += (c.rng.Float64()*2 - 1) * turnRate

	diff := compass - c.Heading
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	c.Heading = wrapHeading(c.Heading + 0.05*diff)

	c.Heading = wrapHeading(c.Heading)

	dist := c.SpeedMS * dt.Seconds()

	c.Pos.X += dist * math.Sin(c.Heading)
	c.Pos.Y += dist * math.Cos(c.Heading)
}

// wrapHeading brings an angle back into [0, 2π).
func wrapHeading(h float64) float64 {
	h = math.Mod(h, 2*math.Pi)
	if h < 0 {
		h += 2 * math.Pi
	}
	return h
}

func (c *Cow) TurnAway(p float64) bool {
	if c.rng.Float64() > p*c.Trained {
		return false
	}

	c.Heading = wrapHeading(c.Heading + math.Pi + (c.rng.Float64()*2-1)*turnSpread)
	return true
}
