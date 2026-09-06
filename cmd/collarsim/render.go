package main

import (
	"fmt"
	"strings"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
)

// Debug view only. Delete this file once the browser map arrives in Loop 2.

const (
	cols   = 64
	rows   = 26
	margin = 15.0 // metres of ground shown outside the fence
)

// render clears the screen and draws the paddock, the cow's recent trail and
// its current state as ASCII art.
func render(f fence.Rect, trail []geom.Point, state fence.State, edgeM float64) {
	if len(trail) == 0 {
		return
	}

	minX, maxX := f.MinX-margin, f.MaxX+margin
	minY, maxY := f.MinY-margin, f.MaxY+margin

	cell := func(p geom.Point) (int, int, bool) {
		c := int((p.X - minX) / (maxX - minX) * cols)
		r := int((maxY - p.Y) / (maxY - minY) * rows)
		if c < 0 || c >= cols || r < 0 || r >= rows {
			return 0, 0, false
		}
		return c, r, true
	}

	grid := make([][]rune, rows)
	for r := range grid {
		grid[r] = []rune(strings.Repeat(" ", cols))
	}

	put := func(p geom.Point, ch rune) {
		if c, r, ok := cell(p); ok {
			grid[r][c] = ch
		}
	}

	stepX := (maxX - minX) / cols / 2
	stepY := (maxY - minY) / rows / 2
	for x := f.MinX; x <= f.MaxX; x += stepX {
		put(geom.Point{X: x, Y: f.MinY}, '-')
		put(geom.Point{X: x, Y: f.MaxY}, '-')
	}
	for y := f.MinY; y <= f.MaxY; y += stepY {
		put(geom.Point{X: f.MinX, Y: y}, '|')
		put(geom.Point{X: f.MaxX, Y: y}, '|')
	}

	for _, p := range trail {
		put(p, '.')
	}

	cow := trail[len(trail)-1]
	mark := 'o'
	switch state {
	case fence.Warning:
		mark = 'W'
	case fence.Breached:
		mark = 'X'
	}
	put(cow, mark)

	var b strings.Builder
	b.WriteString("\033[H\033[2J")
	b.WriteString("+" + strings.Repeat("-", cols) + "+\n")
	for _, row := range grid {
		b.WriteString("|" + string(row) + "|\n")
	}
	b.WriteString("+" + strings.Repeat("-", cols) + "+\n")
	b.WriteString(fmt.Sprintf(" x=%.1f  y=%.1f   %-9s edge=%.1fm   trail=%d\n",
		cow.X, cow.Y, state, edgeM, len(trail)))
	fmt.Print(b.String())
}
