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
	"github.com/veerbal1/paddock/internal/shared/mqttx"
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

	mc, err := mqttx.New(*broker, "collarsim")
	if err != nil {
		log.Fatalf("broker: %v", err)
	}
	defer mc.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The device fleet's own fence copy. Step 3 replaces this seed value
	// with the retained fence the broker hands over on subscribe.
	f := fence.Rect{MinX: 0, MaxX: 100, MinY: 0, MaxY: 100}
	s := sim.New(*herd, *seed, geom.Point{X: 50, Y: 50}, f)
	s.Speed = *speed

	go s.Run(ctx)

	log.Printf("collarsim: farm %s, %d cows, broker %s", *farmID, *herd, *broker)

	// Run closes the ping channel when ctx is done, so this loop is the
	// shutdown signal too: it ends once the last tick has been published.
	failed := 0
	for ping := range s.Pings() {
		if err := mc.PublishPing(*farmID, ping); err != nil {
			failed++
		}
	}
	log.Printf("collarsim: %d pings failed to publish, exiting", failed)
}
