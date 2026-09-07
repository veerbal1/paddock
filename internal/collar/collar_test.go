package collar

import (
	"testing"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

func testFence() fence.Rect {
	return fence.Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}
}

// TestCollarObserve walks one cow out through the fence and back in. The order
// matters: unlike the fence tests, each step depends on the state left behind
// by the one before it.
func TestCollarObserve(t *testing.T) {
	c := New("cow-01", testFence(), 10)

	steps := []struct {
		name        string
		p           geom.Point
		wantState   telemetry.State
		wantChanged bool
		wantCue     telemetry.Cue
	}{
		{"starts inside", geom.Point{X: 100, Y: 100}, telemetry.Inside, false, telemetry.CueNone},
		{"enters warning", geom.Point{X: 195, Y: 100}, telemetry.Warning, true, telemetry.CueAudio},
		{"stays in warning", geom.Point{X: 193, Y: 100}, telemetry.Warning, false, telemetry.CueNone},
		// Pushing further out does not restart the ladder, so no fresh cue here.
		{"breaches", geom.Point{X: 205, Y: 100}, telemetry.Breached, true, telemetry.CueNone},
		{"stays breached", geom.Point{X: 210, Y: 100}, telemetry.Breached, false, telemetry.CueNone},
		{"back to warning", geom.Point{X: 195, Y: 100}, telemetry.Warning, true, telemetry.CueNone},
		{"back inside", geom.Point{X: 100, Y: 100}, telemetry.Inside, true, telemetry.CueNone},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			obs := c.Observe(s.p)
			if obs.To != s.wantState {
				t.Errorf("Observe(%v).To = %v, want %v", s.p, obs.To, s.wantState)
			}
			if obs.Changed != s.wantChanged {
				t.Errorf("Observe(%v).Changed = %v, want %v", s.p, obs.Changed, s.wantChanged)
			}
			if obs.Cue != s.wantCue {
				t.Errorf("Observe(%v).Cue = %v, want %v", s.p, obs.Cue, s.wantCue)
			}
		})
	}
}

// TestCollarEscalates parks a cow in the warning zone and records which tick
// each cue fired on. It checks the ladder climbs in order and, just as
// importantly, that the collar goes quiet after the pulse instead of shocking
// a cow that clearly cannot or will not move.
func TestCollarEscalates(t *testing.T) {
	c := New("cow-01", testFence(), 10)
	warningPoint := geom.Point{X: 195, Y: 100}

	type fired struct {
		tick int
		cue  telemetry.Cue
	}

	var got []fired
	for tick := 0; tick < 40; tick++ {
		if obs := c.Observe(warningPoint); obs.Cue != telemetry.CueNone {
			got = append(got, fired{tick, obs.Cue})
		}
	}

	want := []fired{
		{0, telemetry.CueAudio},
		{escalateAfter, telemetry.CueVibration},
		{escalateAfter * 2, telemetry.CuePulse},
	}

	if len(got) != len(want) {
		t.Fatalf("fired %d cues over 40 ticks, want %d: got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cue %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestCollarNoCueWhenReturning covers the rule that makes cue compliance
// meaningful: a cow walking back in is doing what we asked, so it must not be
// cued for crossing the same zone on the way home.
func TestCollarNoCueWhenReturning(t *testing.T) {
	c := New("cow-01", testFence(), 10)

	if obs := c.Observe(geom.Point{X: 205, Y: 100}); obs.To != telemetry.Breached {
		t.Fatalf("setup: To = %v, want breached", obs.To)
	}

	obs := c.Observe(geom.Point{X: 195, Y: 100})
	if obs.To != telemetry.Warning {
		t.Fatalf("To = %v, want warning", obs.To)
	}
	if obs.Cue != telemetry.CueNone {
		t.Errorf("returning cow was cued with %v, want none", obs.Cue)
	}
}

// TestCollarLadderRestartsAfterReturning proves the reset is real: once a cow
// has come all the way back inside, heading out again starts at audio rather
// than resuming where the previous escalation left off.
func TestCollarLadderRestartsAfterReturning(t *testing.T) {
	c := New("cow-01", testFence(), 10)
	warningPoint := geom.Point{X: 195, Y: 100}

	for tick := 0; tick < 20; tick++ {
		c.Observe(warningPoint) // climbs to at least vibration
	}

	c.Observe(geom.Point{X: 100, Y: 100}) // all the way back inside

	if obs := c.Observe(warningPoint); obs.Cue != telemetry.CueAudio {
		t.Errorf("cue after returning = %v, want audio", obs.Cue)
	}
}
