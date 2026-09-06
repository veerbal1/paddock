package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/veerbal1/paddock/internal/collar"
	"github.com/veerbal1/paddock/internal/cow"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := fence.Rect{MinX: 0, MaxX: 100, MinY: 0, MaxY: 100}
	c := cow.New("cow-01", geom.Point{X: 50, Y: 50}, 42)
	col := collar.New("cow-01", f, 10)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	trail := make([]geom.Point, 0, 300)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("shutting down")
			return

		case <-ticker.C:
			c.Step(time.Second)
			state, _ := col.Observe(c.Pos)
			d := f.DistanceToEdgeM(c.Pos)

			trail = append(trail, c.Pos)
			if len(trail) > 300 {
				trail = trail[1:]
			}
			render(f, trail, state, d)
		}
	}
}
