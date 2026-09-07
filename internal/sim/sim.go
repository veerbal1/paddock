package sim

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/veerbal1/paddock/internal/cow"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

type Sim struct {
	cows  []*cow.Cow
	pings chan telemetry.Ping
}

const spread = 20.0
const pingBuffer = 256

func (s *Sim) Pings() <-chan telemetry.Ping {
	return s.pings
}

func New(n int, seed int64, start geom.Point) *Sim {
	master := rand.New(rand.NewSource(seed))

	cows := make([]*cow.Cow, 0, n)
	for i := 0; i < n; i++ {
		p := geom.Point{
			X: start.X + (master.Float64()*2-1)*spread,
			Y: start.Y + (master.Float64()*2-1)*spread,
		}
		cows = append(cows, cow.New(fmt.Sprintf("cow-%02d", i+1), p, master.Int63()))
	}

	return &Sim{cows: cows, pings: make(chan telemetry.Ping, pingBuffer)}
}

func (s *Sim) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, c := range s.cows {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					fmt.Println("Close triggered")
					return
				case <-ticker.C:
					c.Step(time.Second)
					s.pings <- telemetry.Ping{CowID: c.ID, Pos: c.Pos, At: time.Now()}
					// fmt.Printf("%s  x=%.1f y=%.1f\n", c.ID, c.Pos.X, c.Pos.Y)
				}
			}
		}()
	}

	wg.Wait()
	close(s.pings)
}
