package sim

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/veerbal1/paddock/internal/collar"
	"github.com/veerbal1/paddock/internal/cow"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/geom"
	"github.com/veerbal1/paddock/internal/telemetry"
)

type Sim struct {
	units []*unit
	pings chan telemetry.Ping
}

const spread = 40.0
const pingBuffer = 256

// unit is one cow with the collar strapped to it. They live and tick together.
type unit struct {
	cow    *cow.Cow
	collar *collar.Collar
}

func (s *Sim) Pings() <-chan telemetry.Ping {
	return s.pings
}

func New(n int, seed int64, start geom.Point, f fence.Rect) *Sim {
	master := rand.New(rand.NewSource(seed))

	units := make([]*unit, 0, n)
	for i := 0; i < n; i++ {
		p := geom.Point{
			X: start.X + (master.Float64()*2-1)*spread,
			Y: start.Y + (master.Float64()*2-1)*spread,
		}
		id := fmt.Sprintf("cow-%02d", i+1)
		c := cow.New(id, p, master.Int63())
		col := collar.New(id, f, 10)
		units = append(units, &unit{cow: c, collar: col})
	}

	return &Sim{units: units, pings: make(chan telemetry.Ping, pingBuffer)}
}

func (s *Sim) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, u := range s.units {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					u.cow.Step(time.Second)
					obs := u.collar.Observe(u.cow.Pos)
					s.pings <- telemetry.Ping{
						CowID: u.cow.ID,
						Pos:   u.cow.Pos,
						At:    time.Now(),
						State: obs.To,
						Cue:   obs.Cue,
					}
				}
			}
		}()
	}

	wg.Wait()
	close(s.pings)
}
