package telemetry

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/shared/geom"
)

// TestPingRoundTrip is the contract test between the two binaries: whatever
// collarsim marshals, backend must unmarshal back to the same values. The
// custom Marshal/Unmarshal pairs on State, Cue and Activity are exactly the
// kind of code that fails one-way and silently — publish works, subscribe
// quietly reads zero.
func TestPingRoundTrip(t *testing.T) {
	for _, st := range []State{Inside, Warning, Breached, Escaped} {
		for _, cue := range []Cue{CueNone, CueAudio, CueVibration, CuePulse} {
			for _, act := range []Activity{Grazing, Walking, Resting} {
				want := Ping{
					V:        1,
					CowID:    "cow-07",
					Pos:      geom.Point{X: 12.5, Y: -3.25},
					At:       time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
					State:    st,
					Cue:      cue,
					Activity: act,
				}

				raw, err := json.Marshal(want)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var got Ping
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got != want {
					t.Errorf("round trip changed the ping:\n got %+v\nwant %+v\nwire %s", got, want, raw)
				}
			}
		}
	}
}

// TestFarmIDNeverTravels: the farm is the broker-seen topic's business. A
// collar that could claim its own farm could claim someone else's.
func TestFarmIDNeverTravels(t *testing.T) {
	raw, err := json.Marshal(Ping{V: 1, FarmID: "f1", CowID: "cow-07"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "f1") {
		t.Errorf("farm id leaked onto the wire: %s", raw)
	}
}

// TestStateOrderingIsLoadBearing guards the assumption the collar's cue
// ladder is built on. Reordering these constants would break escalation
// with no compile error anywhere.
func TestStateOrderingIsLoadBearing(t *testing.T) {
	if !(Inside < Warning && Warning < Breached && Breached < Escaped) {
		t.Fatal("state order changed: the collar's cue ladder compares states with < and >")
	}
	if !(CueNone < CueAudio && CueAudio < CueVibration && CueVibration < CuePulse) {
		t.Fatal("cue order changed: the ladder climbs by adding 1 to the level")
	}
}
