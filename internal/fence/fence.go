package fence

import "github.com/veerbal1/paddock/internal/geom"

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
