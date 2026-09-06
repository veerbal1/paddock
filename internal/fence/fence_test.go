package fence

import (
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
