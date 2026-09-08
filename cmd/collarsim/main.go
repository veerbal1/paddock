package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/server"
	"github.com/veerbal1/paddock/internal/sim"
)

func main() {
	speed := flag.Int("speed", 1, "physics steps per real second")
	seed := flag.Int64("seed", 42, "master random seed")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := fence.Rect{MinX: 0, MaxX: 100, MinY: 0, MaxY: 100}
	s := sim.New(50, *seed, geom.Point{X: 50, Y: 50}, f)
	s.Speed = *speed
	b := backend.New()

	go s.Run(ctx)

	srv := &http.Server{Addr: ":8080", Handler: server.New(b).Routes()}
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("server: %v", err)
			stop()
		}
	}()

	b.Consume(s.Pings())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		log.Printf("shutdown: %v", err)
	}
}
