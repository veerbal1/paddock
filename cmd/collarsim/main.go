package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/sim"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s := sim.New(50, 42, geom.Point{X: 50, Y: 50})
	b := backend.New()

	go s.Run(ctx)

	b.Consume(s.Pings())
}
