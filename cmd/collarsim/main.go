package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/veerbal1/paddock/internal/edge/sim"
	"github.com/veerbal1/paddock/internal/shared/fence"
	"github.com/veerbal1/paddock/internal/shared/geom"
)

// env reads a setting from the environment, falling back to a default.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		speed  = flag.Int("speed", 1, "physics steps per real second")
		seed   = flag.Int64("seed", 42, "master random seed")
		herd   = flag.Int("herd", 50, "number of cows")
		broker = flag.String("broker", env("PADDOCK_BROKER", "tcp://localhost:1883"), "MQTT broker address")
		farmID = flag.String("farm", env("PADDOCK_FARM", "1"), "farm this fleet belongs to")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The device fleet's own fence copy. Step 3 replaces this seed value
	// with the retained fence the broker hands over on subscribe.
	f := fence.Rect{MinX: 0, MaxX: 100, MinY: 0, MaxY: 100}
	s := sim.New(*herd, *seed, geom.Point{X: 50, Y: 50}, f)
	s.Speed = *speed

	// One line per collar, each with its own id and its own will. This is
	// the expensive-looking choice that makes a single collar able to die.
	if err := s.Connect(*broker, *farmID); err != nil {
		log.Fatalf("broker: %v", err)
	}
	defer s.Disconnect()

	go s.Run(ctx)

	log.Printf("collarsim: farm %s, %d collars, %d connections, broker %s",
		*farmID, *herd, *herd, *broker)

	// Every collar publishes on its own radio inside Run. This loop only
	// drains the simulator's observation tap so Run never blocks on it, and
	// it ends when Run closes the channel — which is the shutdown signal.
	for range s.Pings() {
	}
	log.Printf("collarsim: %d pings failed to publish, exiting", s.Failed())
}
