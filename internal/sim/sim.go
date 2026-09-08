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
	Speed int
	Clock Clock

	// fence is the single source of truth for the paddock bounds. The
	// HTTP handler writes it via SetFence while Run reads it every tick,
	// so every access takes mu. Collar copies are updated under the same
	// lock so one tick never mixes old and new bounds.
	mu    sync.RWMutex
	fence fence.Rect
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

// SetFence replaces the paddock bounds on every collar at once. It takes
// effect on the next tick at the latest: Run holds mu for the whole compute
// phase, so a fence can never land halfway through the herd.
func (s *Sim) SetFence(f fence.Rect) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fence = f
	for _, u := range s.units {
		u.collar.Fence = f
	}
}

// Fence returns the current bounds.
func (s *Sim) Fence() fence.Rect {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fence
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

	return &Sim{units: units, pings: make(chan telemetry.Ping, pingBuffer), Speed: 1, Clock: RealClock{}, fence: f}
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
			// Compute under mu so SetFence can't land mid-herd. Sends
			// happen after unlock: the ping channel can block, and a lock
			// must never cover a blocking send.
			s.mu.Lock()
			pings := make([]telemetry.Ping, 0, len(s.units))
			for _, u := range s.units {
				for i := 0; i < s.Speed; i++ {
					u.cow.Step(time.Second, center)
				}
				obs := u.collar.Observe(u.cow.Pos)

				if obs.Cue != telemetry.CueNone {
					u.cow.TurnAway(cueResponse(obs.Cue))
				}

				pings = append(pings, telemetry.Ping{
					CowID:    u.cow.ID,
					Pos:      u.cow.Pos,
					At:       s.Clock.Now(),
					State:    obs.To,
					Cue:      obs.Cue,
					Activity: u.cow.Activity,
				})
			}
			s.mu.Unlock()

			for _, ping := range pings {
				select {
				case s.pings <- ping:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
