package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/veerbal1/paddock/internal/cloud/backend"
	"github.com/veerbal1/paddock/internal/cloud/server"
	"github.com/veerbal1/paddock/internal/cloud/store"
	"github.com/veerbal1/paddock/internal/shared/fence"
	"github.com/veerbal1/paddock/internal/shared/mqttx"
	"github.com/veerbal1/paddock/internal/shared/telemetry"
)

// fenceHolder is the cloud-side copy of the fence: what the owner authored.
// It satisfies server.FenceSetter so main never imports sim.
// Device-side copies live in collarsim; Step 3 syncs them via retained MQTT.
type fenceHolder struct {
	mu sync.RWMutex
	f  fence.Rect
}

func (h *fenceHolder) SetFence(f fence.Rect) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.f = f
}

func (h *fenceHolder) Fence() fence.Rect {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.f
}

// defaultFence is only ever seen by a farm that has never drawn one. Any
// farm with a fence on disk gets it back at boot — see hydrateFence.
var defaultFence = fence.Rect{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100}

// env reads a setting from the environment, falling back to a default.
// Flags win over env, env wins over the default — compose sets env, a
// developer overrides with a flag.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		addr      = flag.String("addr", env("PADDOCK_ADDR", ":8080"), "owner API listen address")
		broker    = flag.String("broker", env("PADDOCK_BROKER", "tcp://localhost:1883"), "MQTT broker address")
		dsn       = flag.String("dsn", env("PADDOCK_DSN", "postgres://paddock:paddock@localhost:5432/paddock?sslmode=disable"), "Postgres DSN")
		farmID    = flag.String("farm", env("PADDOCK_FARM", "1"), "farm this backend serves")
		paddockID = flag.String("paddock", env("PADDOCK_PADDOCK", "p1"), "paddock the fence belongs to")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	mc, err := mqttx.New(*broker, "backend")
	if err != nil {
		log.Fatalf("broker: %v", err)
	}

	b := backend.New().WithStore(st)
	holder := &fenceHolder{f: defaultFence}

	// Disk is read exactly here, at boot. Everything after this point
	// answers from memory, so history has to be put back before the first
	// request or the first ping arrives.
	hydrateAlerts(ctx, b, st, *farmID)
	hydrateFence(ctx, holder, st, mc, *farmID, *paddockID)

	srv := &http.Server{
		Addr:    *addr,
		Handler: server.New(b, holder).WithDownlink(st, mc, *farmID, *paddockID).Routes(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
			stop()
		}
	}()

	// Ingest. The subscribe callback runs on Paho's own goroutine, so it
	// never blocks: a full buffer drops the ping and counts it. QoS 0 said
	// the same thing at the protocol level — a lost position is replaced a
	// second later.
	var dropped atomic.Int64
	pings := make(chan telemetry.Ping, 256)
	if err := mc.Subscribe(*farmID, func(p telemetry.Ping) {
		select {
		case pings <- p:
		default:
			dropped.Add(1)
		}
	}); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	// Collar liveness. This is a status, not an alert: short dropouts are
	// normal in a paddock, and a policy for "dark too long" belongs in
	// Loop 5. For now it is recorded in the log only — nothing persists it,
	// so "since when" is still an open piece of Step 3.
	if err := mc.SubscribeStatus(*farmID, func(collarID, status string) {
		log.Printf("collar %s: %s", collarID, status)
	}); err != nil {
		log.Fatalf("subscribe status: %v", err)
	}

	drainCtx, abandonDrain := context.WithCancel(context.Background())
	defer abandonDrain()
	consumed := make(chan struct{})
	go func() {
		defer close(consumed)
		b.Consume(drainCtx, pings)
	}()

	log.Printf("backend: farm %s, api %s, broker %s", *farmID, *addr, *broker)
	<-ctx.Done()

	// Shutdown order is the whole point: stop taking new work, then stop
	// the source, then let what we already accepted finish reaching disk.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("shutdown: %v", err)
	}
	mc.Close()
	close(pings)

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		abandonDrain()
		log.Printf("shutdown: drain timed out, %d pings may be unsaved", len(pings))
	}
	if n := dropped.Load(); n > 0 {
		log.Printf("backend: dropped %d pings on a full buffer", n)
	}
}

// hydrateAlerts replays episodes from disk into memory. A failure here is
// loud but not fatal: a backend with no history still protects cows, and
// refusing to boot would be the worse outage.
func hydrateAlerts(ctx context.Context, b *backend.Backend, st *store.Store, farmID string) {
	rows, err := st.ListAlerts(ctx, farmID, backend.MaxAlerts)
	if err != nil {
		log.Printf("hydrate: alerts: %v", err)
		return
	}

	// store.Alert is the row, backend.Alert is the domain type. Main is the
	// composition root, so the translation between the two lives here and
	// neither package has to know about the other.
	alerts := make([]backend.Alert, len(rows))
	for i, r := range rows {
		alerts[i] = backend.Alert{
			CowID:     r.CowID,
			State:     r.State,
			StartedAt: r.StartedAt,
			EndedAt:   r.EndedAt,
		}
	}
	b.Hydrate(alerts)
	log.Printf("hydrate: %d alert episodes restored", len(alerts))
}

// hydrateFence puts the authored fence back and re-pins it on the board.
// The broker holds a retained copy already, but only until the broker
// itself restarts — the diary is the durable one, so boot makes the board
// agree with it rather than trusting it.
func hydrateFence(ctx context.Context, h *fenceHolder, st *store.Store, mc *mqttx.Client, farmID, paddockID string) {
	v, raw, ok, err := st.LatestFence(ctx, farmID, paddockID)
	if err != nil {
		log.Printf("hydrate: fence: %v", err)
		return
	}
	if !ok {
		log.Printf("hydrate: no fence on record, using default %+v", defaultFence)
		return
	}

	var f fence.Rect
	if err := json.Unmarshal(raw, &f); err != nil {
		log.Printf("hydrate: fence v%d unreadable: %v", v, err)
		return
	}
	h.SetFence(f)
	if err := mc.PublishFence(farmID, paddockID, v, raw); err != nil {
		log.Printf("hydrate: re-pin v%d failed: %v", v, err)
	}
	log.Printf("hydrate: fence v%d restored %+v", v, f)
}
