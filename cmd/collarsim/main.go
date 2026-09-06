package main

import (
	"fmt"
	"time"

	"github.com/veerbal1/paddock/internal/collar"
	"github.com/veerbal1/paddock/internal/cow"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

func main() {
	f := fence.Rect{MinX: 0, MaxX: 200, MinY: 0, MaxY: 200}
	c := cow.New("cow-01", geom.Point{X: 198, Y: 100}, 42)
	col := collar.New("cow-01", f, 10)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.Step(time.Second)
		state, _ := col.Observe(c.Pos)

		d := f.DistanceToEdgeM(c.Pos)
		fmt.Printf("x=%.1f y=%.1f  %-8s  edge=%.1fm\n", c.Pos.X, c.Pos.Y, state, d)
	}

}
