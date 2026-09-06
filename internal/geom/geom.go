package geom

// Point is a position on the farm's local plane, in metres from the
// farm origin. It is not a GPS coordinate: lat/lng is converted to a
// Point only at the system boundary.
type Point struct {
	X float64 // metres east of origin
	Y float64 // metres north of origin
}
