package collar

import (
	"testing"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

func TestCollarObserve(t *testing.T) {
	f := fence.Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}
	c := New("cow-01", f, 10)

	steps := []struct {
		name        string
		p           geom.Point
		wantState   telemetry.State
		wantChanged bool
	}{
		{name: "starts inside", p: geom.Point{X: 100, Y: 100}, wantState: telemetry.Inside, wantChanged: false},
		{name: "enters warning", p: geom.Point{X: 195, Y: 100}, wantState: telemetry.Warning, wantChanged: true},
		{name: "stays in warning", p: geom.Point{X: 193, Y: 100}, wantState: telemetry.Warning, wantChanged: false},
		{name: "breaches", p: geom.Point{X: 205, Y: 100}, wantState: telemetry.Breached, wantChanged: true},
		{name: "stays breached", p: geom.Point{X: 210, Y: 100}, wantState: telemetry.Breached, wantChanged: false},
		{name: "back to warning", p: geom.Point{X: 195, Y: 100}, wantState: telemetry.Warning, wantChanged: true},
		{name: "back inside", p: geom.Point{X: 100, Y: 100}, wantState: telemetry.Inside, wantChanged: true},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			gotState, gotChanged := c.Observe(s.p)
			if gotState != s.wantState {
				t.Errorf("Observe(%v) state = %v, want %v", s.p, gotState, s.wantState)
			}
			if gotChanged != s.wantChanged {
				t.Errorf("Observe(%v) changed = %v, want %v", s.p, gotChanged, s.wantChanged)
			}
		})
	}
}
