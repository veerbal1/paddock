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
)

// Cow is a simulated animal. Pos is its true position - only the
// simulator knows this. A collar reads it through GPS, with error.
type Cow struct {
	ID      string
	Pos     geom.Point
	rng     *rand.Rand
	Heading float64 // in radian, cow's face direction
	SpeedMS float64
}

// New returns a cow at start with its own seeded random source, so the
// same seed always produces the same path.
func New(id string, start geom.Point, seed int64) *Cow {
	rng := rand.New(rand.NewSource(seed))
	return &Cow{ID: id, Pos: start, rng: rng, Heading: rng.Float64() * 2 * math.Pi, SpeedMS: 1.0}
}

// Step advances the cow by one tick of simulated time dt.
func (c *Cow) Step(dt time.Duration) {
	c.Heading += (c.rng.Float64()*2 - 1) * turnRate

	c.Heading = math.Mod(c.Heading, 2*math.Pi)

	if c.Heading < 0 {
		c.Heading += 2 * math.Pi
	}

	dist := c.SpeedMS * dt.Seconds()

	c.Pos.X += dist * math.Sin(c.Heading)
	c.Pos.Y += dist * math.Cos(c.Heading)
}
