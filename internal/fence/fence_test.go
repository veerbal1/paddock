package fence

import (
	"math"
	"testing"

	"github.com/veerbal1/paddock/internal/geom"
)

func TestRectContains(t *testing.T) {
	r := Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}

	tests := []struct {
		name string
		p    geom.Point
		want bool
	}{
		{name: "inside", p: geom.Point{X: 100, Y: 100}, want: true},
		{name: "outside#01", p: geom.Point{X: 201, Y: 100}, want: false},
		{name: "outside#02", p: geom.Point{X: 100, Y: 201}, want: false},
		{name: "outside#03", p: geom.Point{X: -1, Y: 100}, want: false},
		{name: "outside#04", p: geom.Point{X: 100, Y: -1}, want: false},
		{name: "on edge#01", p: geom.Point{X: 200, Y: 100}, want: true},
		{name: "on edge#02", p: geom.Point{X: 100, Y: 200}, want: true},
		{name: "on edge#03", p: geom.Point{X: 0, Y: 100}, want: true},
		{name: "on edge#04", p: geom.Point{X: 100, Y: 0}, want: true},
		{name: "on corner#01", p: geom.Point{X: 201, Y: 201}, want: false},
		{name: "on corner#02", p: geom.Point{X: -1, Y: -1}, want: false},
		{name: "on corner#03", p: geom.Point{X: 200, Y: -1}, want: false},
		{name: "on corner#04", p: geom.Point{X: -1, Y: 200}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Contains(tt.p)
			if got != tt.want {
				t.Errorf("Contains(%v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

func TestRectDistanceToEdgeM(t *testing.T) {
	r := Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}

	tests := []struct {
		name string
		p    geom.Point
		want float64
	}{
		{
			name: "center",
			p: geom.Point{
				X: 100,
				Y: 100,
			},
			want: 100,
		},
		{
			name: "near east edge",
			p: geom.Point{
				X: 195,
				Y: 100,
			},
			want: 5,
		},
		{
			name: "near west edge",
			p: geom.Point{
				X: 5,
				Y: 100,
			},
			want: 5,
		},
		{
			name: "near north edge",
			p: geom.Point{
				X: 100,
				Y: 195,
			},
			want: 5,
		},
		{
			name: "near south edge",
			p: geom.Point{
				X: 100,
				Y: 5,
			},
			want: 5,
		},
		{
			name: "on east edge",
			p: geom.Point{
				X: 200,
				Y: 100,
			},
			want: 0,
		},
		{
			name: "outside east",
			p: geom.Point{
				X: 205,
				Y: 100,
			},
			want: -5,
		},
		{
			name: "outside west",
			p: geom.Point{
				X: -5,
				Y: 100,
			},
			want: -5,
		},
		{
			name: "outside north",
			p: geom.Point{
				X: 100,
				Y: 205,
			},
			want: -5,
		},
		{
			name: "outside south",
			p: geom.Point{
				X: 100,
				Y: -5,
			},
			want: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.DistanceToEdgeM(tt.p)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("DistanceToEdgeM(%v) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

func TestRectEvaluate(t *testing.T) {
	r := Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}
	const warnM = 10

	tests := []struct {
		name string
		p    geom.Point
		want State
	}{
		{name: "deep inside", p: geom.Point{X: 100, Y: 100}, want: Inside},
		{name: "just inside warning zone", p: geom.Point{X: 191, Y: 100}, want: Warning},
		{name: "just outside warning zone", p: geom.Point{X: 189, Y: 100}, want: Inside},
		{name: "on edge", p: geom.Point{X: 200, Y: 100}, want: Warning},
		{name: "outside", p: geom.Point{X: 205, Y: 100}, want: Breached},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Evaluate(tt.p, warnM)
			if got != tt.want {
				t.Errorf("Evaluate(%v, %v) = %v, want %v", tt.p, warnM, got, tt.want)
			}
		})
	}
}
