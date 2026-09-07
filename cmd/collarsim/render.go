package main

import (
	"fmt"
	"strings"

	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

// Debug view only. Delete once the browser map arrives in Loop 2.

const (
	cols   = 64
	rows   = 26
	margin = 15.0 // metres of ground drawn outside the fence
)

// render clears the screen and draws the paddock with every cow on it.
func render(f fence.Rect, pings []telemetry.Ping, cues map[telemetry.Cue]int) {
	minX, maxX := f.MinX-margin, f.MaxX+margin
	minY, maxY := f.MinY-margin, f.MaxY+margin

	grid := make([][]rune, rows)
	for r := range grid {
		grid[r] = []rune(strings.Repeat(" ", cols))
	}

	put := func(p geom.Point, ch rune) {
		c := int((p.X - minX) / (maxX - minX) * float64(cols))
		r := int((maxY - p.Y) / (maxY - minY) * float64(rows))
		if c >= 0 && c < cols && r >= 0 && r < rows {
			grid[r][c] = ch
		}
	}

	stepX := (maxX - minX) / float64(cols) / 2
	stepY := (maxY - minY) / float64(rows) / 2
	for x := f.MinX; x <= f.MaxX; x += stepX {
		put(geom.Point{X: x, Y: f.MinY}, '-')
		put(geom.Point{X: x, Y: f.MaxY}, '-')
	}
	for y := f.MinY; y <= f.MaxY; y += stepY {
		put(geom.Point{X: f.MinX, Y: y}, '|')
		put(geom.Point{X: f.MaxX, Y: y}, '|')
	}

	// Draw in order of severity so that when two cows share a cell the one
	// worth looking at is the one you see.
	counts := map[telemetry.State]int{}
	for _, p := range pings {
		counts[p.State]++
	}
	for _, want := range []telemetry.State{telemetry.Inside, telemetry.Warning, telemetry.Breached, telemetry.Escaped} {
		for _, p := range pings {
			if p.State != want {
				continue
			}
			switch want {
			case telemetry.Warning:
				put(p.Pos, 'W')
			case telemetry.Breached, telemetry.Escaped:
				put(p.Pos, 'X')
			default:
				put(p.Pos, 'o')
			}
		}
	}

	var b strings.Builder
	b.WriteString("\033[H\033[2J")
	b.WriteString("+" + strings.Repeat("-", cols) + "+\n")
	for _, row := range grid {
		b.WriteString("|" + string(row) + "|\n")
	}
	b.WriteString("+" + strings.Repeat("-", cols) + "+\n")

	fmt.Fprintf(&b, " herd %3d    o inside %3d    W warning %3d    X breached %3d\n",
		len(pings),
		counts[telemetry.Inside],
		counts[telemetry.Warning],
		counts[telemetry.Breached]+counts[telemetry.Escaped])
	fmt.Fprintf(&b, " cues total  audio %4d      vibration %4d    pulse %4d\n",
		cues[telemetry.CueAudio], cues[telemetry.CueVibration], cues[telemetry.CuePulse])

	fmt.Print(b.String())
}
