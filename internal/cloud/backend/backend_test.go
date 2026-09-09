package backend

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/veerbal1/paddock/internal/shared/geom"
	"github.com/veerbal1/paddock/internal/shared/telemetry"
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
	b.Consume(context.Background(), ch)
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

// fakeStore records what reached disk, so the persistence path is provable
// without Postgres. That is the whole reason Store is an interface.
type fakeStore struct {
	saves  []telemetry.Ping
	opened []string
	closed []string
	failOn string // cow whose SavePing should fail
}

func (f *fakeStore) SavePing(_ context.Context, _ string, p telemetry.Ping) error {
	if p.CowID == f.failOn {
		return errors.New("disk is on fire")
	}
	f.saves = append(f.saves, p)
	return nil
}

func (f *fakeStore) OpenAlert(_ context.Context, _, cowID string, _ telemetry.State, _ time.Time) (int64, error) {
	f.opened = append(f.opened, cowID)
	return int64(len(f.opened)), nil
}

func (f *fakeStore) CloseAlert(_ context.Context, _, cowID string, _ time.Time) (bool, error) {
	f.closed = append(f.closed, cowID)
	return true, nil
}

// TestConsumeWritesEpisodesToStore pins the contract between the episode
// machine and disk: one OpenAlert when she leaves, one CloseAlert when she
// returns — not one per ping.
func TestConsumeWritesEpisodesToStore(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	fs := &fakeStore{}
	b := New().WithStore(fs)

	ch := make(chan telemetry.Ping, 8)
	for i, st := range []telemetry.State{
		telemetry.Inside, telemetry.Breached, telemetry.Breached, telemetry.Inside,
	} {
		ch <- ping("cow-01", t0.Add(time.Duration(i)*time.Second), st)
	}
	close(ch)
	b.Consume(context.Background(), ch)

	if len(fs.saves) != 4 {
		t.Errorf("SavePing calls = %d, want 4 (every ping)", len(fs.saves))
	}
	if len(fs.opened) != 1 || fs.opened[0] != "cow-01" {
		t.Errorf("OpenAlert calls = %v, want one for cow-01", fs.opened)
	}
	if len(fs.closed) != 1 || fs.closed[0] != "cow-01" {
		t.Errorf("CloseAlert calls = %v, want one for cow-01", fs.closed)
	}
}

// TestConsumeSurvivesStoreFailure: a database that refuses a write must not
// stop the herd being tracked. Memory stays authoritative for the API.
func TestConsumeSurvivesStoreFailure(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	fs := &fakeStore{failOn: "cow-01"}
	b := New().WithStore(fs)

	ch := make(chan telemetry.Ping, 2)
	ch <- ping("cow-01", t0, telemetry.Breached)
	ch <- ping("cow-02", t0, telemetry.Breached)
	close(ch)
	b.Consume(context.Background(), ch)

	if got := b.Alerts(); len(got) != 2 {
		t.Fatalf("episodes = %d, want 2 despite the failed write", len(got))
	}
}

// TestConsumeDrainsOnCancel proves the shutdown promise: a ping already
// buffered when SIGTERM lands still reaches disk.
func TestConsumeDrainsOnCancel(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	fs := &fakeStore{}
	b := New().WithStore(fs)

	ch := make(chan telemetry.Ping, 4)
	ch <- ping("cow-01", t0, telemetry.Inside)
	ch <- ping("cow-02", t0, telemetry.Inside)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already shutting down before Consume even looks
	b.Consume(ctx, ch)

	if len(fs.saves) != 2 {
		t.Errorf("saved %d buffered pings on cancel, want 2", len(fs.saves))
	}
}

// TestHydrateRestoresOpenEpisodes is the restart test in miniature: a cow
// who was out when the process died must have her return close the episode
// she already has, not open a second one.
func TestHydrateRestoresOpenEpisodes(t *testing.T) {
	t0 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	ended := t0.Add(time.Minute)

	b := New()
	b.Hydrate([]Alert{
		{CowID: "cow-05", State: telemetry.Inside, StartedAt: t0, EndedAt: &ended},
		{CowID: "cow-09", State: telemetry.Breached, StartedAt: t0.Add(2 * time.Minute)},
	})

	if got := b.Alerts(); len(got) != 2 {
		t.Fatalf("episodes after hydrate = %d, want 2", len(got))
	}

	ch := make(chan telemetry.Ping, 1)
	ch <- ping("cow-09", t0.Add(3*time.Minute), telemetry.Inside)
	close(ch)
	b.Consume(context.Background(), ch)

	got := b.Alerts()
	if len(got) != 2 {
		t.Fatalf("episodes = %d, want still 2 — her return opened a duplicate", len(got))
	}
	if got[1].Open() {
		t.Error("cow-09 came home but her episode is still open")
	}
	if got[1].EndedAt == nil || !got[1].EndedAt.Equal(t0.Add(3*time.Minute)) {
		t.Errorf("closed at %v, want +3m", got[1].EndedAt)
	}
}
