package cow

import (
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/shared/geom"
	"github.com/veerbal1/paddock/internal/shared/telemetry"
)

// TestActivitySplit checks every mode is actually visited over a long run.
// Bounds are deliberately loose: with p=0.001 there are only ~30 switches
// in 30000 ticks, so shares swing wildly between seeds. This guards against
// a stuck mode (never leaves grazing), not exact percentages.
func TestActivitySplit(t *testing.T) {
	c := New("cow-01", geom.Point{X: 100, Y: 100}, 42, 1.0)
	center := geom.Point{X: 100, Y: 100}

	counts := map[telemetry.Activity]int{}
	const steps = 30000
	for i := 0; i < steps; i++ {
		c.Step(time.Second, center)
		counts[c.Activity]++
	}

	for _, a := range []telemetry.Activity{telemetry.Grazing, telemetry.Walking, telemetry.Resting} {
		share := float64(counts[a]) / steps
		if share < 0.05 || share > 0.90 {
			t.Errorf("activity %v share = %.3f, want 0.05–0.90 (counts=%v)", a, share, counts)
		}
	}
}
