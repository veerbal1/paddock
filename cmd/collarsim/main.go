package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
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

	go s.Run(ctx)

	// TODO Step 1: publish pings to broker here instead of dropping.
	n := 0
	for range s.Pings() {
		n++
	}
	log.Printf("collarsim: drained %d pings, exiting", n)
}
