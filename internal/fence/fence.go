package fence

import "github.com/veerbal1/paddock/internal/geom"

// State is the current state of the fence.
type State string

const Inside State = "inside"
const Breached State = "breached"

const Warning State = "warning"

// Rect is a rectangle on the farm's local plane, in metres from the
// farm origin. They are parallel to the farm's axes. Not diagonally or random.
type Rect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// Contains reports whether p lies inside the rectangle.
func (r Rect) Contains(p geom.Point) bool {
	return !(p.X > r.MaxX || p.X < r.MinX || p.Y > r.MaxY || p.Y < r.MinY)
}

// DistanceToEdgeM returns metres from p to the nearest edge.
// Negative means p is outside.
func (r Rect) DistanceToEdgeM(p geom.Point) float64 {
	west := p.X - r.MinX
	east := r.MaxX - p.X

	north := r.MaxY - p.Y
	south := p.Y - r.MinY

	return min(west, east, north, south)
}

// Evaluate returns the state of p relative to the fence. warnM is how
// many metres inside the edge the warning zone begins.
func (r Rect) Evaluate(p geom.Point, warnM float64) State {
	d := r.DistanceToEdgeM(p)
	if d < 0 {
		return Breached
	}

	if d <= warnM {
		return Warning
	}

	return Inside
}
