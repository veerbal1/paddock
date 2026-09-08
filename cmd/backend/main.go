package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/server"
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	b := backend.New()
	holder := &fenceHolder{f: fence.Rect{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100}}

	srv := &http.Server{Addr: ":8080", Handler: server.New(b, holder).Routes()}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
			stop()
		}
	}()

	// TODO Step 1: subscribe to broker here, feed b via channel.
	// TODO Step 2: Store (Postgres) replaces in-memory backend behind same shape.
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("shutdown: %v", err)
	}
}
