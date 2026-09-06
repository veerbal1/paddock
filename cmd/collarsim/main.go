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
	c := cow.New("cow-01", geom.Point{X: 198, Y: 197}, 42)
	col := collar.New("cow-01", f)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		c.Step(time.Second)
		state, _ := col.Observe(c.Pos)

		fmt.Printf("x=%.1f y=%.1f  %s\n", c.Pos.X, c.Pos.Y, state)
	}

}
