package backend

import (
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

// feed runs Consume over scripted pings and returns what it recorded.
// Consume returns when the channel closes, so this is synchronous.
func feed(pings []telemetry.Ping) *Backend {
	b := New()
	ch := make(chan telemetry.Ping, len(pings))
	for _, p := range pings {
		ch <- p
	}
	close(ch)
	b.Consume(ch)
	return b
}

func ping(cow string, at time.Time, st telemetry.State) telemetry.Ping {
	return telemetry.Ping{CowID: cow, Pos: geom.Point{}, At: at, State: st}
}

func TestAlertOpensAndCloses(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	b := feed([]telemetry.Ping{
		ping("cow-01", t0, telemetry.Inside),
		ping("cow-01", t0.Add(time.Second), telemetry.Warning),
		ping("cow-01", t0.Add(2*time.Second), telemetry.Breached),
		ping("cow-01", t0.Add(3*time.Second), telemetry.Breached), // still out: no new episode
		ping("cow-01", t0.Add(4*time.Second), telemetry.Inside),
	})

	got := b.Alerts()
	if len(got) != 1 {
		t.Fatalf("episodes = %d, want 1", len(got))
	}
	a := got[0]
	if a.CowID != "cow-01" || !a.StartedAt.Equal(t0.Add(time.Second)) {
		t.Fatalf("opened %+v, want cow-01 at +1s", a)
	}
	if a.EndedAt == nil || !a.EndedAt.Equal(t0.Add(4*time.Second)) {
		t.Fatalf("closed %+v, want end at +4s", a)
	}
	if a.Open() {
		t.Fatal("episode still open after cow came back")
	}
}

func TestAlertOpensOnFirstSeenOutside(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	b := feed([]telemetry.Ping{
		ping("cow-07", t0, telemetry.Breached),
	})

	got := b.Alerts()
	if len(got) != 1 || !got[0].Open() || got[0].CowID != "cow-07" {
		t.Fatalf("episodes = %+v, want 1 open for cow-07", got)
	}
}

func TestNoAlertWhenAlwaysInside(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	b := feed([]telemetry.Ping{
		ping("cow-02", t0, telemetry.Inside),
		ping("cow-02", t0.Add(time.Second), telemetry.Inside),
	})

	if got := b.Alerts(); len(got) != 0 {
		t.Fatalf("episodes = %+v, want none", got)
	}
}
