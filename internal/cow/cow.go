package cow

import (
	"math/rand"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
)

// Cow is a simulated animal. Pos is its true position - only the
// simulator knows this. A collar reads it through GPS, with error.
type Cow struct {
	ID  string
	Pos geom.Point
	rng *rand.Rand
}

// New returns a cow at start with its own seeded random source, so the
// same seed always produces the same path.
func New(id string, start geom.Point, seed int64) *Cow {
	rng := rand.New(rand.NewSource(seed))
	return &Cow{ID: id, Pos: start, rng: rng}
}

// Step advances the cow by one tick of simulated time dt.
func (c *Cow) Step(dt time.Duration) {
	randomX := (c.rng.Float64()*2 - 1) // -1 or 1
	randomY := (c.rng.Float64()*2 - 1) // -1 or 1

	c.Pos.X += randomX * dt.Seconds()
	c.Pos.Y += randomY * dt.Seconds()
}
