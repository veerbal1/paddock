package sim

import (
	"context"
	"fmt"
	"math/rand"
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
	Speed int
	Clock Clock
}

// Clock is the source of ping timestamps. Production uses RealClock;
// tests use FakeClock. Nothing here calls time.Now() directly, so
// --speed can never fast-forward a timestamp by accident.
type Clock interface{ Now() time.Time }

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

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

func (s *Sim) Centroid() geom.Point {
	xSum := 0.0
	ySum := 0.0
	var length float64 = float64(len(s.units))
	for _, unit := range s.units {
		xSum += unit.cow.Pos.X
		ySum += unit.cow.Pos.Y
	}

	return geom.Point{
		X: xSum / length,
		Y: ySum / length,
	}
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
		trained := 0.9 + master.Float64()*0.1 // 0.9 – 1.0
		if master.Float64() < 0.1 {
			trained = 0.3 // 10% untrained
		}
		c := cow.New(id, p, master.Int63(), trained)
		col := collar.New(id, f, 10)
		units = append(units, &unit{cow: c, collar: col})
	}

	return &Sim{units: units, pings: make(chan telemetry.Ping, pingBuffer), Speed: 1, Clock: RealClock{}}
}

// cueResponse is how likely a well-trained cow is to turn away from each cue.
// The cow's own Trained factor scales this further.
func cueResponse(c telemetry.Cue) float64 {
	switch c {
	case telemetry.CueAudio:
		return 0.85
	case telemetry.CueVibration:
		return 0.92
	case telemetry.CuePulse:
		return 0.98
	}
	return 0
}

func (s *Sim) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer close(s.pings)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			center := s.Centroid()
			for _, u := range s.units {
				for i := 0; i < s.Speed; i++ {
					u.cow.Step(time.Second, center)
				}
				obs := u.collar.Observe(u.cow.Pos)

				if obs.Cue != telemetry.CueNone {
					u.cow.TurnAway(cueResponse(obs.Cue))
				}

				ping := telemetry.Ping{
					CowID:    u.cow.ID,
					Pos:      u.cow.Pos,
					At:       s.Clock.Now(),
					State:    obs.To,
					Cue:      obs.Cue,
					Activity: u.cow.Activity,
				}
				select {
				case s.pings <- ping:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
