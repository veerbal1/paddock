package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
	go srv.ListenAndServe()

	b.Consume(s.Pings())
}
