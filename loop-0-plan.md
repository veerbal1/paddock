# Loop 0 — One cow, one terminal

**Detailed learning plan.** Companion to `virtual-fencing-go-roadmap.md`. Read that file's Loop 0 section once for the *spec*; this file is the *path* through it.

**Goal:** prove you can model the domain in plain Go before any infrastructure exists.
**Time:** 2–4 days, 2–4 hour sessions.
**End state:** `go run .` shows one believable cow wandering a real paddock, printing `WARNING` near the fence and `BREACH` outside it, with tests that prove the geometry is right and the path is reproducible.

---

## How to use this document

Every step has the same three beats:

- **Understand** — the idea, and *why it shows up here and not somewhere else*. Read this before touching the keyboard.
- **Build** — signatures and structure. Not implementations.
- **Prove** — the test that shows you actually got it. Written before or with the code, never after.

Each step opens with the *pain left over from the previous step*. That pain is the reason the step exists. If a step ever feels arbitrary, go back and re-read the pain — you probably skipped it.

### Who writes what

The roadmap is explicit on this and it matters most in Loop 0, because Loop 0 is almost entirely "the hard 20%":

- **You write:** haversine, the meters↔degrees conversion, the movement function, the correlated random walk, the fence check. All of it. By hand, first, wrong the first time.
- **Claude writes:** scaffolding, `Makefile`, `.gitignore`, test *cases* (not the code under test), critique of what you wrote.

The rule of thumb: if you'd be asked about it in an interview, you type it. Nobody will ever ask you about your `Makefile`.

---

## The through-line

Loop 0 is a chain, not a checklist. Each link exists because the previous link left something broken:

```
  a cow is somewhere            →  you need a point                      →  Step 1
  how far is it from there?     →  lat/lng deltas are meaningless        →  haversine
  haversine measures, can't move→  you need the inverse: meters→degrees  →  Step 2
  moving needs a direction      →  fresh random heading = jitter, not a cow → Step 3
  now it's random               →  random means untestable               →  Step 4 (seeded RNG)
  and it calls time.Now()       →  hidden global input, also untestable  →  Step 5 (clock)
  now it's fully deterministic  →  so you can snapshot it                →  Step 6 (golden test)
  a reproducible cow + a box    →  distance-to-edge reuses Step 2's math →  Step 7 (fence)
  three states, printed         →  one cow in one terminal isn't a system→  Loop 1
```

Notice the two closed loops in there — they are the reason Loop 0 hangs together instead of being five unrelated exercises:

1. **Haversine tests the movement code.** Step 2 moves a cow 100 m east; Step 1's haversine measures the result and must read 100 m. The two halves of the geometry check each other.
2. **Determinism enables the golden test.** Steps 4 and 5 exist *only* so Step 6 is possible. They are not general good practice you're adopting on faith — they are load-bearing for a specific test.

---

## Step 0 — Scaffold (30 minutes)

Not learning, just clearing the desk. Let Claude do most of it.

```bash
git init && git add -A && git commit -m "Initial commit: roadmap and module"
```

Create:

```
paddock/
  go.mod                    (exists)
  main.go                   package main — the tick loop and printing
  internal/
    geo/                    Point, Haversine, Move, the constants
    sim/                    Cow, the walk, Clock
    fence/                  Rect, Contains, DistanceToEdge, State
  docs/adr/0001-geometry-and-determinism.md
  NOTES.md
  IDEAS.md
  CLAUDE.md
  Makefile                  run, test, race, lint
  .gitignore
```

Two decisions embedded here, worth understanding rather than copying:

**Why `internal/`?** Go refuses imports of `internal/...` from outside the module. More importantly for you: in Loop 4 the *server-side* geofence processor computes the same containment check as the collar, from the same package. `internal/geo` is shared infrastructure from day one. Putting geometry in `main` now means moving it in Loop 2 and again in Loop 4.

**Why `main.go` at the root?** Because the DoD says `go run .`. In Loop 2 this becomes `cmd/collarsim/main.go` when the process splits in two. Don't pre-build that; just know it's coming.

**Open the ADR draft now.** Title it, write the *Context* section (three sentences: this is Loop 0, you need geometry and reproducibility, everything downstream depends on both), and leave *Decision* and *Consequences* empty. You'll fill them at Step 8. Opening it now is what makes you notice the decisions while you're making them instead of after.

---

## Step 1 — A point, and "how far apart?"

### Understand

A cow is at a place. A place is a `(latitude, longitude)` pair. Two floats — so the obvious move is to treat them like x and y on a grid.

They are not x and y. They are **angles**. Latitude is the angle north or south of the equator; longitude is the angle east or west of the prime meridian. There is no ruler in a coordinate pair, only two rotations.

So the first honest question — *how far apart are two points* — has no answer until you convert angles into a length along the Earth's surface. That's what haversine is for.

Before you look it up, spend two minutes on the naive version, because feeling it fail is the point:

```
dLat = 0.001
dLng = 0.001
dist = sqrt(dLat² + dLng²) = 0.0014...
```

0.0014 *what*? Not meters. Not degrees, exactly — it's a mongrel of two different angles that happen to share a unit name. And the two components aren't even the same physical size, which is the whole subject of Step 2.

### The formula, and what each piece is doing

```
a = sin²(Δφ/2) + cos φ₁ · cos φ₂ · sin²(Δλ/2)
c = 2 · atan2(√a, √(1−a))
d = R · c
```

where φ = latitude in radians, λ = longitude in radians, R = Earth's radius in meters.

Read it as three moves rather than one incantation:

- **`c` is the angle** between the two points as seen from the center of the Earth, in radians. Everything above the last line is just a numerically well-behaved way to compute that angle.
- **`d = R · c`** is arc length. That's the entire idea: distance along a sphere is radius times the angle you swept. If you remember only one thing about haversine, remember this line.
- **`cos φ₁ · cos φ₂`** on the longitude term is the same shrinking factor you'll meet properly in Step 2, appearing here for the first time. Longitude differences count for less the further you are from the equator. Haversine already knows this; your movement code will have to be told.

**Why `atan2(√a, √(1−a))` and not `asin(√a)`?** They're mathematically the same. `atan2` stays accurate for near-antipodal points where `asin` loses precision. You'll never have two cows on opposite sides of the planet, so this doesn't matter for your system — but knowing *why* the standard form is written that way is the difference between having typed a formula and understanding it.

### Build

```go
package geo

const EarthRadiusM = 6371000.0

type Point struct {
    Lat float64 // degrees, positive north
    Lng float64 // degrees, positive east
}

// Haversine returns the great-circle distance between a and b in meters.
func Haversine(a, b Point) float64
```

Two traps:

- **`math.Sin` and `math.Cos` take radians.** Your `Point` is in degrees. Convert at the top of the function, once, and never think about it again. Half of all first-attempt haversines are wrong for exactly this reason.
- **`6371000` is the mean radius.** Earth is an oblate spheroid — flatter at the poles, ~21 km difference between equatorial and polar radius. A sphere is an approximation. Hold that thought; it explains your test tolerances below, and it's the honest answer if someone asks "how accurate is this?"

### Prove

`internal/geo/haversine_test.go`, table-driven. Build it in two tiers, and understand why they're different — this is real test design, not box-ticking.

**Tier 1 — cases derivable from R alone.** These are exact *within your own model*, so the tolerance can be tight (a meter, or less):

| From | To | Expected | Where it comes from |
|---|---|---|---|
| (0, 0) | (0, 0) | 0 m | identity |
| (0, 0) | (0, 1) | 111,195 m | R·π/180, one degree of arc |
| (0, 0) | (1, 0) | 111,195 m | same — and proving these two are equal *at the equator* is the setup for Step 2's punchline |
| (0, 0) | (90, 0) | 10,007,543 m | R·π/2, quarter circumference |
| (-90, 0) | (90, 0) | 20,015,087 m | R·π, half circumference |

Compute those expectations yourself from `R` rather than pasting them. If your haversine and your arithmetic disagree, one of them is wrong and you want to know which.

**Tier 2 — one real-world pair.** Pick two cities, get a reference distance from an independent great-circle calculator, and assert within **0.5%**.

That tolerance is not laziness, and you should be able to say so out loud: haversine models a sphere, the Earth is an ellipsoid, so haversine is off by roughly 0.3–0.5% against a true geodesic (Vincenty/Karney). For a cow in a paddock, 0.5% of 100 m is 50 cm — far below your GPS noise in Loop 2. For a shipping route it would matter. Knowing where the approximation breaks is the point.

Also assert **symmetry**: `Haversine(a,b) == Haversine(b,a)`. Cheap, and it catches a whole class of sign errors.

### Pain left over

You can now *measure* the distance between two points. You still can't *move* a cow. Haversine answers "how far apart are these?" — you need the inverse: "given this point, this direction, and 1.4 meters, where does the cow end up?"

---

## Step 2 — Measuring isn't moving

### Understand

This is the most important step in Loop 0, and the one the roadmap says a virtual-fencing engineer will catch in ten seconds if you get it wrong.

The principle: **cows move in meters; storage and transmission happen in degrees.** So every tick does a round trip — convert the cow's position to a local metric frame, add a displacement in meters, convert back to degrees. Never move a cow by adding to `Lat` and `Lng` directly.

To convert, you need to know how many meters one degree buys you. And here the two axes part ways:

**Latitude.** Lines of latitude are evenly spaced from equator to pole. One degree of latitude is the same arc length everywhere:

```
metersPerDegreeLat = R · π / 180 ≈ 111,195 m
```

**Longitude.** Lines of longitude are *meridians* — they all pass through both poles. They are furthest apart at the equator and converge to a single point at each pole. So one degree of longitude shrinks as you move away from the equator, by exactly the cosine of your latitude:

```
metersPerDegreeLng(φ) = metersPerDegreeLat · cos(φ)
```

| Latitude | Where | cos(φ) | One degree of longitude |
|---|---|---|---|
| 0° | equator | 1.000 | 111,195 m |
| 31°N | Ludhiana | 0.857 | 95,306 m |
| 37°S | Waikato (Halter's farms) | 0.799 | 88,806 m |
| 60°N | Oslo-ish | 0.500 | 55,597 m — exactly half |
| 90° | the pole | 0.000 | 0 m |

**What happens if you skip the cosine.** You compute `dLng = dxMeters / 111195` regardless of latitude. At 31°N the true divisor should be 95,306, so you divide by a number ~17% too large and get a longitude delta ~17% too small. Every eastward step under-moves. The herd's east–west spread compresses by 17% relative to its north–south spread — the whole paddock looks squashed, cows hug the vertical axis, and fence breaches on the east and west edges fire at the wrong distances.

(Note the direction: dividing by too-large a number *under*-moves. The roadmap describes the symptom as "stretched"; whether the map looks stretched or squashed depends on which way you got it wrong. What matters is that the two axes are no longer to the same scale.)

### The heading convention — decide it now, write it down

You are about to introduce an angle that is *not* a latitude or longitude, and this is where people quietly lose an afternoon. Two conventions exist:

- **Compass bearing** — 0° = north, 90° = east, increasing clockwise. `dNorth = d·cos(θ)`, `dEast = d·sin(θ)`.
- **Math angle** — 0° = east, increasing counterclockwise. `dEast = d·cos(θ)`, `dNorth = d·sin(θ)`.

Both are correct. Mixing them sends your cow 90° off. Pick compass bearing (it's what GPS, aviation, and every marine chart use, and it's what you'll want when you print a heading in a log line), put it in a comment on the type, and put it in the ADR.

### Build

```go
// MetersPerDegreeLat is derived from EarthRadiusM so that Move and Haversine
// agree exactly, rather than approximately.
const MetersPerDegreeLat = EarthRadiusM * math.Pi / 180

// MetersPerDegreeLng returns meters per degree of longitude at the given latitude.
func MetersPerDegreeLng(latDeg float64) float64

// Move returns the point reached by travelling distM meters from p along
// headingRad, a compass bearing in radians (0 = north, increasing clockwise).
func Move(p Point, headingRad, distM float64) Point
```

**Derive the constant from `EarthRadiusM`; do not hardcode 111320.** You'll see 111,320 in textbooks — it's a decent ellipsoid average. But your haversine assumes a sphere of radius 6,371,000, and that sphere's degree is 111,195 m. Mixing the two puts a permanent ~0.1% disagreement between your mover and your measurer, which turns the tight round-trip test below into a loose one. One model, one constant, derived.

Use the *current* latitude for the cosine on each step. Over a farm-sized paddock the difference is microscopic, but it's the correct formulation and it costs one function call.

### Prove

`internal/geo/move_test.go`. This is the test where the two halves lock together:

- **Round trip.** `Move(p, north, 100)` then `Haversine(p, result)` ≈ 100 m, to within a centimeter. Repeat for east, south, west, and a few odd bearings. If this passes tightly, your conversion constant and your haversine share one model.
- **Direction.** Moving north increases `Lat` and leaves `Lng` unchanged. Moving east increases `Lng` and leaves `Lat` unchanged. Catches the compass/math mixup instantly.
- **The cosine, explicitly.** Move 1000 m east from (0°, X) and from (60°, X). The *longitude delta* at 60° must be almost exactly twice the delta at the equator — because `cos(60°) = 0.5`. Meanwhile `Haversine` reports 1000 m for both. This single test is the entire concept, executable. Write it and keep it; it's the one you'll point at in an interview.
- **Closure.** Move 100 m north then 100 m south; you should land within a millimeter of where you started.

### Pain left over

You can move a cow one step in a direction you supply. Nothing chooses the direction yet.

---

## Step 3 — A mover needs a direction

### Understand

Try the naive thing first, in your head or in five lines of code, because the failure is instructive: **pick a fresh uniformly-random heading every tick.**

Result: the cow takes a step north, then southwest, then east, then south. Successive steps cancel. The animal vibrates in place and its net displacement grows like √n instead of n. It looks like a dropped marble, not a grazing cow. That's an uncorrelated random walk — Brownian motion — and it's wrong for animals for a physical reason: a cow has mass and momentum and is generally *going somewhere*.

The fix is one word: **correlate consecutive headings.** Don't draw a new heading; perturb the old one.

```
heading[t+1] = heading[t] + turn,   turn ~ Normal(0, σ)
```

That's it. That's the whole "correlated random walk." One parameter, σ, controls how animal-like it looks:

| σ per step | Behaviour |
|---|---|
| 0 | perfectly straight line — a train, not a cow |
| small (a few degrees) | long smooth arcs — a cow walking with purpose |
| moderate (~15–30°) | meandering, plausible grazing |
| very large (~180°) | back to uncorrelated jitter |

Play with σ until it looks right on the printout. "Believable" is the DoD's actual word, and calibrating a parameter by eye against a real-world referent is a legitimate modelling skill, not hand-waving.

**Two behaviours to add, both cheap now:**

- **Speed.** A dairy cow walking is roughly 1.1–1.4 m/s. A grazing cow is far slower — mostly standing, taking short steps, call it 0.1–0.3 m/s average. Over a day on pasture a cow might cover 2–4 km.
- **Modes.** The roadmap's Loop 0 wording is "occasionally stops to graze." The minimum version: a small per-tick probability of switching between a slow grazing speed and a faster walking speed, each with a different σ. Loop 1 promotes this to explicit named states (`grazing` / `walking` / `resting`) with realistic dwell times, reported in every ping, because their collars carry IMU sensors precisely to classify this. Build the two-mode version now and leave the door open.

**Wrap the heading** into `[0, 2π)` so it doesn't drift to ±10⁶ radians over a long run and start losing float precision.

### Build

```go
package sim

type Cow struct {
    ID      string
    Pos     geo.Point
    Heading float64 // compass bearing, radians
    SpeedMS float64
    // rng lives here — see Step 4
}

// Step advances the cow by one simulation tick of duration dt.
func (c *Cow) Step(dt time.Duration)
```

**`dt` is a parameter, not a constant.** Right now you will always pass one second and it will feel like pointless ceremony. It is not. In Loop 1, `--speed=60` works by passing `60 * time.Second` as `dt` while the wall clock still ticks once per real second — *speed scales the physics, never the clock.* If you hardcode "one second of movement per call" here, you will be refactoring the core of the simulator in Loop 1. This one parameter is the cheapest forward-compatibility you'll ever buy.

So: `distance = c.SpeedMS * dt.Seconds()`, then `c.Pos = geo.Move(c.Pos, c.Heading, distance)`.

### Prove

Randomness makes assertions awkward — which is precisely the pain that opens Step 4. For now, test the properties that hold regardless of the draws:

- With σ = 0 and a fixed heading, N steps of 1 m produce a point N meters away. Deterministic, and it proves `Step` composes `Move` correctly.
- Heading always stays in `[0, 2π)`.
- Distance travelled in one step always equals `SpeedMS × dt`, whatever the heading.
- Eyeball it: print 200 steps and look at the shape.

### Pain left over

You just made your simulator random. Run it twice, get two different cows. You cannot write a meaningful assertion about the path, you cannot reproduce a bug you just saw, and you cannot write the golden test the DoD demands.

---

## Step 4 — Determinism, part one: the random source

### Understand

The instinct is `rand.Float64()` from the package level. Don't, for three separate reasons — and it's worth being able to name all three:

1. **It's auto-seeded per process.** Modern Go seeds the global source randomly at startup. Two runs, two different cow paths, by design.
2. **It's shared global state.** Any package in your process — yours, a dependency's — that draws from the global source perturbs *your* sequence. Your reproducibility silently depends on code you don't control.
3. **It doesn't scale to Loop 1.** Fifty cows will be fifty goroutines. A single `*rand.Rand` is not safe for concurrent use; sharing one across goroutines is a data race that `go test -race` will find in Loop 1.

The fix is to make randomness an **explicit, owned dependency** rather than an ambient service. Each cow holds its own generator, seeded from a master seed you control.

This is the same idea as the clock in Step 5, and it's worth seeing them as one principle rather than two tips: *a function that reads hidden global state is not a function you can test.* Randomness and time are the two hidden inputs almost every simulation has. Make both parameters and the whole thing becomes a pure function of `(seed, config, dt)`.

### Build

```go
type Cow struct {
    // ...
    rng *rand.Rand
}
```

**One generator per cow, derived from one master seed.** Don't hand every cow the same seed (they'd move identically — fifty cows in a conga line). Two clean approaches:

- Seed a master generator with the run seed, then draw each cow's seed from it. Adding a cow doesn't disturb the others' streams if you draw in order.
- Or `seed + index`. Simpler; fine for now, and slightly more fragile with some generators.

**One API choice to make deliberately.** Go 1.26 gives you both `math/rand` (the roadmap's `rand.New(rand.NewSource(seed))`) and `math/rand/v2` (`rand.New(rand.NewPCG(s1, s2))`). v2 is the modern API with better generators. Either works.

What matters is that **you pick one and stay**, because Step 6's golden file encodes the exact output stream of whichever generator you chose. Switching later invalidates the golden file — not a disaster, but it should be a decision, not a surprise. Put the choice and this consequence in the ADR.

For the turn angle you want `rng.NormFloat64() * sigma` — a normal distribution, present in both APIs. Normal rather than uniform because small turns should be common and sharp turns rare, which is how animals actually move.

### Prove

- Two cows constructed with the same seed produce identical paths.
- Two cows with different seeds diverge.
- Fifty cows from one master seed all have different paths.

### Pain left over

Same seed, same path — except your positions are timestamped, and every run stamps a different time. One hidden input down, one to go.

---

## Step 5 — Determinism, part two: time (and the three clocks)

### Understand

`time.Now()` is a function that takes no arguments and returns something different every call. It is a hidden global input, exactly like the global RNG, and it makes any code that calls it untestable in the same way.

The fix is the same shape: make it an explicit dependency.

```go
type Clock interface { Now() time.Time }
```

A real implementation returns `time.Now()`. A fake one returns a fixed instant and can be advanced by hand in tests. Production code — and this is the rule the roadmap states flatly — **never calls `time.Now()` directly.**

### The three time concepts — do not let these blur

This is the single biggest source of confusion in the whole project, and Loop 1's `--speed` ADR is where it bites. Three different time-ish things live in your tick loop and they are independent:

| | What it is | Who provides it | Does `--speed` change it? |
|---|---|---|---|
| **Wall clock** | The timestamp stamped on a position report | `Clock.Now()` | **No — never.** Always real wall-clock time |
| **Physics `dt`** | How much simulated movement one `Step` produces | parameter to `Step(dt)` | **Yes.** This is the only thing `--speed` scales |
| **Tick interval** | How often the loop runs in real time | `time.Ticker` | No. Stays at 1 second |

Read the roadmap's Loop 1 sentence with that table in front of you: *"speed scales the physics, never the clock. `--speed=60` means each real-time tick advances the cow's movement by 60 simulated seconds; every timestamp the collar reports is still real wall-clock time."*

**Why it must be this way.** In Loop 4 the ingest service rejects pings timestamped in the future — a real, necessary validation against bad device clocks and spoofing. If `--speed` fast-forwarded your timestamps, your own simulator would be permanently sending pings from the future and ingest would reject every one of them. Every wall-clock threshold downstream — stale-collar minutes, alert cooldowns, retention windows — breaks the same way.

The consequence, also from the roadmap: collar hysteresis dwell time is counted in **ticks**, backend thresholds stay in **wall-clock**, and a simulated grazing day runs in 24 real minutes at speed 60.

**The fake clock is for tests, not for `--speed`.** Keep that straight and Loop 1 costs you an afternoon instead of a week.

### Build

```go
package sim

type Clock interface{ Now() time.Time }

type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now() }

type FakeClock struct{ t time.Time }
func (c *FakeClock) Now() time.Time         { return c.t }
func (c *FakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }
```

Then the tick loop in `main.go`:

```go
ticker := time.NewTicker(time.Second)
for range ticker.C {
    cow.Step(dt)
    fmt.Println(clock.Now(), cow.Pos, ...)
}
```

Wire `--seed`, `--dt`, and your paddock coordinates as flags while you're here. You'll use them constantly. (`--speed` itself is Loop 1's; `--dt` is the honest Loop 0 version of the same knob.)

### Prove

- `FakeClock` returns the same instant until advanced, and advances by exactly what you asked.
- A run driven by a `FakeClock` produces identical timestamps twice.

### Pain left over

None — this is the good kind of step. Seed pinned plus clock pinned means your simulation is now a pure function of its inputs. Which means you can freeze its output.

---

## Step 6 — Freeze it: the golden test

### Understand

A **golden test** (or snapshot test) runs the system, serializes the output, and compares it byte-for-byte against a checked-in reference file. It doesn't assert that the output is *correct* — it asserts that the output *hasn't changed*. Those are different jobs, and both are worth having.

Its value here is specific: your unit tests check the pieces. The golden test checks that the pieces still *compose* into the same behaviour after you refactor. When you reorganize the walk in Loop 1, a green golden test says "you moved code, not behaviour." A red one says "you changed the cow, on purpose or by accident — which?"

It's also the DoD line: *a golden test proves seed 42 produces the identical path twice.*

### Build

```
internal/sim/testdata/golden_seed42.json
```

`testdata/` is special to the Go tool: it's ignored when building packages, so anything in there is test fixture by convention.

Run N steps (200 is plenty) with seed 42 and a `FakeClock` starting at a fixed instant. Serialize each position. Compare to the file.

Add the standard update idiom so regenerating is one command and not a manual copy-paste:

```go
var update = flag.Bool("update", false, "update golden files")
// go test ./internal/sim -run TestGoldenPath -update
```

Regenerating should always be a deliberate act with a diff you read. A golden test you update reflexively is worse than no test.

**Round the coordinates when you serialize** — `%.7f` is about 1.1 cm, far finer than your GPS noise will be in Loop 2. Two reasons:

1. Full float64 formatting makes diffs unreadable when the test does fail, and an unreadable diff is a test you'll stop trusting.
2. Portability. You're on `darwin/arm64`; GitHub Actions runs `linux/amd64`. Transcendental functions (`sin`, `cos`, `atan2`) can differ in the last bit or two across architectures and libm implementations. Rounding to 7 decimals is cheap insurance against a golden test that passes on your laptop and fails in CI for reasons that have nothing to do with your code.

### Prove

Run the test twice — green both times. Then deliberately change σ by 1%, watch it go red, and read the diff. A golden test you've never seen fail is a golden test you don't know works.

---

## Step 7 — The fence

### Understand

You now have a reproducible cow. Give it something to be inside of.

Loop 0's fence is a **rectangle in lat/lng** — `minLat, maxLat, minLng, maxLng`. Loop 2 replaces it with arbitrary GeoJSON polygons via `paulmach/orb`; Loop 5 pushes it into PostGIS. The rectangle is a deliberate placeholder, and it teaches the shape of the problem without the polygon machinery.

**Containment is trivial. Proximity is not.** That asymmetry is the lesson of this step.

```
inside  ⟺  minLat ≤ lat ≤ maxLat  ∧  minLng ≤ lng ≤ maxLng
```

Four comparisons, no geometry. But `WARNING` when the cow is *within 10 meters of the edge* is a question about **meters**, and your box is defined in **degrees** — so you're back in Step 2's territory. Compute the perpendicular distance to each of the four edges, in meters, using the same conversion:

```
to north edge:  (maxLat − lat) · MetersPerDegreeLat
to south edge:  (lat − minLat) · MetersPerDegreeLat
to east  edge:  (maxLng − lng) · MetersPerDegreeLng(lat)
to west  edge:  (lng − minLng) · MetersPerDegreeLng(lat)
```

Take the minimum. A negative value means the cow is outside on that side, and its magnitude is how far out.

**Use the same conversion functions from `internal/geo`, not a fresh copy.** If the fence check and the movement code ever disagree about how big a degree is, you get a cow that walks "10 m" and the fence thinks it moved 9.8 m — and you will lose hours to it. One model, one constant, one place.

### Build

```go
package fence

type State int
const (
    Inside State = iota
    Warning
    Breached
)

type Rect struct{ MinLat, MaxLat, MinLng, MaxLng float64 }

// DistanceToEdgeM returns meters to the nearest edge; negative when outside.
func (r Rect) DistanceToEdgeM(p geo.Point) float64

// Evaluate returns Inside, Warning (within warnM of an edge), or Breached.
func (r Rect) Evaluate(p geo.Point, warnM float64) State
```

**A note on what you're building, so it doesn't feel throwaway.** This three-state function is the seed of the collar state machine that carries the rest of the project:

- **Loop 1** turns it into a real state machine — `inside → warning → breached → escaped` — with audio, vibration, and pulse cues logged at each escalation, and cows that turn away from the boundary in response.
- **Loop 2** adds GPS noise, and this function starts *flapping*: inside, outside, inside, sixty breach alerts in an hour from one cow standing still. The cure is hysteresis and dwell time — the warning zone starts *inside* the boundary, a breach requires being clearly outside for a sustained period, and returning to inside requires being clearly inside.
- **Loop 4** makes it mint breach *episodes*, which become the deduplication key that stops Loop 6 from sending the same Telegram message twice.

Right now it returns an enum. Know where it's going.

### Prove

- Center of the box → `Inside`.
- A point 5 m inside the north edge → `Warning`. 15 m inside → `Inside`.
- A point just outside → `Breached`, with a small negative distance.
- **Corners.** Near a corner two edges are both close. Assert the minimum is what you expect. Corners are where naive implementations go wrong.
- **Consistency with `Move`.** Put a cow exactly 20 m inside an edge, `Move` it 15 m toward that edge, and assert the state flipped to `Warning`. This ties Step 7 back to Step 2 the same way Step 2 tied back to Step 1.

---

## Step 8 — Wire it up and close the loop

### The demo

`main.go`: parse flags, build the cow, build the fence, tick once a second, print position and state. Print a line only when the state *changes* — plus a heartbeat every N ticks — or your terminal is a waterfall and you can't see the breaches.

**Practical tip for the demo.** Pick a small paddock (100 m × 100 m) or start the cow near an edge. At ~1 m/s in a 300 m × 300 m field, a correlated random walk can take a long time to find a boundary, and you'll be sitting there watching a contented cow. You want a breach in the first minute or two.

### Close it out

1. **Definition of Done** — copy the checklist below into a GitHub issue and tick every line literally. The roadmap is blunt that half-done DoDs are where later gaps come from.
2. **Explain test** — answer all three out loud. A stumble means you don't own it yet.
3. **Finish the ADR** — the Decision and Consequences sections you left empty at Step 0.
4. **`NOTES.md`** — what broke, what surprised you, what you'd do differently. This is interview-story material, not bookkeeping. The best Loop 0 entries are usually the radians/degrees bug or the moment the round-trip test caught a wrong constant.
5. **Tag it** — `git tag loop-0`.

### The ADR — what actually goes in it

`docs/adr/0001-geometry-and-determinism.md`. Six decisions, each one sentence of *why*:

| Decision | The one-line rationale |
|---|---|
| Spherical Earth, R = 6,371,000 m | ~0.5% error vs. ellipsoid, two orders of magnitude below GPS noise; revisit if the system ever needs sub-meter accuracy |
| `MetersPerDegreeLat` derived from R, not 111320 | one model everywhere; makes `Move`/`Haversine` agree to millimeters instead of ~0.1% |
| Move in meters, convert with cos(latitude) | the axes are not the same size; skipping it distorts every map |
| Compass bearing for headings | matches GPS/aviation convention and reads correctly in logs |
| `math/rand` vs `math/rand/v2` — whichever you chose | golden files encode the generator's stream; switching later invalidates them |
| `Clock` injected; `Step` takes `dt` | both hidden inputs become parameters; `dt` is where Loop 1's `--speed` plugs in without a refactor |

---

## Definition of Done

- [ ] `go run .` shows a believable cow wandering and occasionally breaching
- [ ] `haversine_test.go` passes table-driven tests against known city-pair distances
- [ ] A golden test proves seed 42 produces the identical path twice
- [ ] `go test -race ./...` is green
- [ ] `docs/adr/0001-geometry-and-determinism.md` is complete
- [ ] `NOTES.md` has a Loop 0 entry
- [ ] Tagged `loop-0`

---

## Explain test

Answer out loud, not in your head. These are literally interview questions.

**1. Why can't you compare raw lat/lng deltas to a meter threshold?**
A good answer covers: they're angles, not lengths; the two axes have different scales; the longitude scale is itself a function of latitude, so the same delta means different distances at different places. Bonus: the naive Pythagorean form isn't wrong by a constant you could calibrate away — it's wrong by a factor that changes as the cow moves.

**2. Why is a longitude degree shorter than a latitude degree?**
Meridians converge at the poles; parallels don't. The factor is exactly cos(latitude): 111 km at the equator, 95 km at Ludhiana, 89 km in the Waikato, zero at the pole. Latitude degrees are constant because the spacing between parallels doesn't depend on where you are. Name the failure mode too — a map whose axes are to different scales.

**3. What makes a random walk "correlated," and why does it look more realistic?**
Each heading is the previous heading plus a small perturbation, instead of a fresh independent draw. Physically it's momentum: animals don't reverse direction every second. Statistically, an uncorrelated walk's steps cancel and net displacement grows like √n; a correlated one covers ground like a real animal. σ is the single knob from straight line to jitter.

**A fourth one, not in the roadmap, that you should also be able to answer:** *why does the simulation need a seeded RNG and an injectable clock?* Because a function that reads hidden global state can't be tested, reproduced, or debugged from a report. Make both explicit parameters and the simulation becomes a pure function of its inputs — which is what makes the golden test possible at all.

---

## What Loop 0 hands to Loop 1

The roadmap's handoff line: *hands over the cow model, correct geometry (haversine, cos-latitude), a seeded RNG and injectable clock. Pain that motivates Loop 1: one cow in one terminal isn't a system.*

Concretely, four things you built here slot straight into Loop 1 without modification — which is exactly why they were shaped this way:

| Loop 0 artifact | What Loop 1 does with it |
|---|---|
| `Step(dt time.Duration)` | `--speed=60` passes `60s` as `dt`; the physics scales and the wall clock doesn't. No refactor. |
| One `*rand.Rand` per cow | Fifty cows become fifty goroutines. Per-cow generators mean `go test -race` stays green. |
| `Clock` interface | Timestamps stay real wall-clock time under `--speed`, which is what keeps Loop 4's future-ping validation honest. |
| `fence.State` enum | Grows into the `inside → warning → breached → escaped` escalation machine with cue responses. |

And the pain, precisely: one cow printing to stdout is not a system. Loop 1 makes it fifty cows as goroutines fanning into one channel, a real collar state machine with cues the cows respond to, herd cohesion, and an HTTP API — all in one process, so that Loop 2 has something worth splitting in two.

---

## Trap index

Everything above that will actually bite, in one place:

1. `math.Sin`/`math.Cos` take **radians**; `Point` is in **degrees**.
2. Deriving `MetersPerDegreeLat` from a different Earth model than `Haversine` uses. One R, one constant.
3. Forgetting `cos(latitude)` on the longitude conversion — the ten-second tell.
4. Mixing compass bearing and math angle. Pick one, comment it, ADR it.
5. Using the package-level `rand` instead of an owned `*rand.Rand`.
6. Sharing one `*rand.Rand` across cows — invisible now, a data race in Loop 1.
7. Calling `time.Now()` anywhere outside `RealClock`.
8. Hardcoding one second of movement inside `Step` instead of taking `dt`.
9. Confusing the fake clock with `--speed`. The clock is for tests; `dt` is for speed.
10. Serializing full-precision floats into the golden file — unreadable diffs, and possible CI-vs-laptop drift.
11. Duplicating the degree↔meter conversion inside the fence package instead of importing `geo`.
