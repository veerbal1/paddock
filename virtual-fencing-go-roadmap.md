# Virtual Fencing System in Go — Spiral Learning Roadmap

**The project:** a realistic, reliable simulation of a Halter-style virtual fencing platform. Simulated cows wearing simulated collars move across real GPS coordinates. Collars enforce virtual fence boundaries locally (like real firmware does), stream telemetry over the internet to a cloud backend, and the backend detects breaches, tracks the herd, and alerts the owner with locations. Everything except the animal and the plastic is real.

**The method:** spiral learning. Every loop produces a *complete working system* end to end. The next loop doesn't add a new "phase" — it revisits the same system and deepens one dimension: transport, durability, geospatial depth, observability, orchestration, cloud. You are never stuck in tutorial-land; you always have something running that you can demo.

**Rules of the spiral:**

1. Do not skip ahead. The pain you feel in loop N is what makes the tool introduced in loop N+1 make sense. Feeling the pain first is the point.
2. Each loop has a **Definition of Done** and an **Explain test** — questions you should be able to answer out loud, because these are literally interview questions.
3. Commit from day one. Write a short `NOTES.md` entry at the end of each loop: what broke, what you'd do differently. This becomes your blog post / cover letter material later.
4. Rough pacing at ~10–15 focused hours/week: Loops 0–2 ≈ 3 weeks, Loops 3–5 ≈ 5–6 weeks, Loops 6–8 ≈ 6–7 weeks, Loops 9–10 ≈ 3–4 weeks. Total ≈ 4–5 months from zero. **Coming from the OPD project, expect ≈ 3–4 months** — see the calibration section below. Faster is fine; slower is fine. Skipping "done" criteria is not.

**Core stack (introduced gradually, not all at once):**

| Layer | Choice | Why |
|---|---|---|
| Language | Go (latest stable) | The whole point |
| Device transport | MQTT over the internet | What real cellular collars (Nofence-style) actually use; QoS, retained messages, and last-will map perfectly to collar problems |
| Event backbone | NATS JetStream | Operationally simple (one binary), low latency, built for edge-to-cloud IoT, and it has a built-in MQTT gateway — broker + stream in one system. Kafka is the heavier tool for multi-team/long-retention shops; you'll touch it in Loop 10 so you can speak to both |
| Database | PostgreSQL + PostGIS + TimescaleDB | PostGIS gives you `ST_Contains` geofencing and spatial indexes; Timescale gives you hypertables for ping history. Both are resume lines |
| Live-state cache | **Valkey** (Redis fork; arrives Loop 6) | Latest cow positions + pub/sub fan-out for the live feed; the honest "non-relational database" checkbox. Valkey over Redis because Halter runs Valkey clusters in production — they publish their own Go Kubernetes operator for it |
| Geometry in Go | `paulmach/orb` | Point-in-polygon, haversine distance, GeoJSON — the standard Go geo library |
| APIs | REST + WebSocket (owner-facing), gRPC (simulator control plane — arrives Loop 7) | Mirrors real backend shops |
| Infra | Docker Compose → Kubernetes (kind/k3d) → AWS + Terraform | Matches the exact keywords in Halter-type job specs |
| Observability | slog, Prometheus, Grafana, OpenTelemetry | Non-negotiable for "not a toy" |
| CI | GitHub Actions + golangci-lint + testcontainers-go | Professional hygiene |

---

## How to work a loop — the day-to-day method

The roadmap is a spec, not a tutorial: it names landmarks, you build the road. Nothing in it is code, and most of what you'll type isn't named here at all — that's the point.

**Per loop, in order:**

1. Read the current loop and the next loop's line in the handoff chain. Don't reread the whole file; it's a reference.
2. Plan for an hour before code: data model, package layout, API shapes, build order. Open the ADR *draft* now (context, options) and finish it (decision, consequences) at the end — starting it first makes you notice the decision before you accidentally make it.
3. Build in the stated order, runnable at every step; failing test or chaos scenario *before* the fix.
4. Copy the Definition of Done into a GitHub issue and tick every line literally. Half-done DoDs are where later gaps come from.
5. Answer the Explain test out loud. A stumble means you don't own it yet — cheaper to learn now than in an interview.
6. Write `NOTES.md`, finish the ADR, tag the commit `loop-N`, and stop for the day.

**Per session:** 2–4 hours. Start by re-reading the current DoD; end with a green build and a commit. Never leave the tree broken overnight.

**New things — three kinds, three rules:** things the roadmap names → build them your way. Things it doesn't name but you need (a config loader, a fixture generator, a retry helper) → build them without asking. Your own feature ideas → write them in `IDEAS.md` and keep going; do one now only if it directly serves the *current* DoD, otherwise it waits for a flourish slot or Loop 10. Scope creep kills solo projects and always arrives disguised as a good idea.

**AI tools — the one rule:** let Claude Code scaffold boilerplate, generate test cases, explain options, and review PRs. But write the hard 20% yourself first — the collar state machine, the geometry, the outbox worker, the episode reconciliation, the scaling-vs-ordering choice — because those are what you'll be asked about, and the only way to own an answer is to have made the mistakes. Ask the AI to *critique* your version rather than produce one; that's where learning compounds, and your `CLAUDE.md` notes write themselves.

**When to ask for help:** with code and a specific question, an ADR you want challenged, or after two days stuck. Not "is the plan right" — that's settled — but "here's my state machine, poke holes in it."

---

## What Halter's public GitHub tells you (and what to do about it)

Their two orgs — `halter-corp` (mostly forks of things they build on) and `halter` (their open-source work) — leak more architecture than any job ad. Read the forks as a bill of materials: a company forks what it depends on. This table was built from the org landing pages only (repo names, upstreams, languages, recency) — not from reading their code — so each row carries a confidence tag. Treat **high** as fact, **medium** as a strong guess to verify in conversation, and never assert a medium-confidence item to a Halter engineer as if you know it.

| What's there | What it means | Confidence | What you do |
|---|---|---|---|
| Forks of **amazon-freertos** and **esp-afr-sdk** | Collar firmware is built on Amazon FreeRTOS for Espressif silicon (high). That SDK's *default* path is **MQTT to AWS IoT Core with per-device X.509 certificates** — but a LoRa-tower fleet may well terminate MQTT on their own brokers instead (medium) | High / Medium | Your MQTT choice is confirmed correct, and mTLS device certs become the *default* in Loop 9, not an option |
| Fork of **nanopb** (protobuf in C, small code size) + **ts-protoc-gen** (protobuf → TypeScript/gRPC-Web) | Protobuf is their wire format end to end, device through web — two independent forks at opposite ends of the wire | High | Loop 10's protobuf swap moves from "optional flourish" to strongly recommended |
| **valkey-cluster-operator** — their own Go Kubernetes operator, actively maintained | They run Valkey (Redis fork) on Kubernetes and write their own controllers in Go | High (original code, updated Aug 2026) | Use Valkey, not Redis, in Loop 6. And see the Loop 10 stretch goal below |
| **terraform-provider-kubepatch**, **terraform-provider-utils** (their own, Go), plus Terraform provider/resource forks | They write custom Terraform providers in Go — infra tooling is real engineering there, not YAML-wrangling | High | Reinforces Loop 9; the Terraform work is not a checkbox |
| Concourse CI resources (**github-pr-resource**, **terraform-resource**) | Their CI was, and may still be, Concourse — CI stacks migrate and these could be legacy | Medium-Low | Keep GitHub Actions (portable, free), but know the difference and say so if asked |
| **go-unifi**, **docker-openwisp**, Ansible **community.network** | The tower fleet likely runs UniFi hardware managed with OpenWISP and Ansible — the Platform Communications team's world | Medium | Out of scope for you (no towers), but it explains their "fleet automation / Ansible" JD lines |
| Forks of **BMI270** and **BNO080** IMU drivers | Collars carry accelerometer/gyro sensors (high); that these drive grazing/rumination classification is inferred from their ML job ads (medium) | High / Medium | Add an activity state to your simulator (below) |
| Fork of **lorawan-stack** (The Things Stack, Go) | Their LoRaWAN network server is Go | High | Context only |
| Fork of **openhtf** (Google's hardware test framework) | Manufacturing-line test rigs | High | Context only |

Two engineers appear across both orgs and are visibly the platform/infra people; when Loop 10 says to send the demo link to engineers rather than the jobs inbox, their public profiles are a reasonable starting point.

**To go deeper than I did** (an evening, well spent before the Loop 10 email): read `valkey-cluster-operator` and `terraform-provider-utils` end to end — their own code shows their Go conventions, error handling, test style, and how they structure a reconcile loop, which is exactly what your operator stretch goal should imitate; check commit activity on the Concourse forks to settle whether it's live or legacy; and skim `terraform-provider-kubepatch` to see what gap in stock Terraform they felt strongly enough to fill.

---

## Calibration: coming from the OPD project

You arrive with Go fundamentals from OPD, so this roadmap's "Learn" lists are partly behind you. The rule: **still do every loop's build and Definition of Done** (the *system* needs each piece), but **skip the study time** for anything you already shipped in OPD. What's known versus new, by loop — apply the "known" column only for OPD loops you actually completed:

| Loop | Already known if you finished OPD Loop… | What's genuinely new here |
|---|---|---|
| 0 | 0: modules, structs, tests, `time.Ticker` | geometry: haversine, degree↔meter conversion, correlated random walk |
| 1 | 0: `context`, graceful shutdown, `slog`, `net/http`, race detector | many-producer fan-in over channels, simulation clock with time-scale, deterministic seeded runs |
| 2 | 0–1: pgx, goose, Compose, HTTP server, testcontainers | HTTP *client* resilience, polygon geometry, GPS noise + hysteresis, Leaflet |
| 3 | — | MQTT is entirely new: QoS, retained, LWT, topic design |
| 4 | 3: outbox, idempotency, at-least-once thinking | JetStream itself; event time vs processing time; out-of-order replay |
| 5 | 5: RLS, partitioning, `EXPLAIN` discipline | PostGIS, Timescale hypertables/continuous aggregates, spatial indexes |
| 6 | 1: auth/cookies; 3: outbox + SKIP LOCKED; 2: optimistic locking; 6: Redis→Valkey (same API) | WebSocket (vs SSE), React console, device credentials + MQTT topic ACLs |
| 7 | 4: Prometheus, OTel, pprof, k6 | gRPC/protobuf, load-generating over 10k MQTT connections |
| 8 | 7: k3s, probes, HPA, rolling deploys | exposing MQTT (TCP, not HTTP) through Kubernetes; NATS/Valkey operators |
| 9 | 1: TLS, backups (Caddy/VPS) | Terraform, AWS, SNS/SQS/S3, mTLS device identity |
| 10 | 6: Kafka | the bridge design and the JetStream↔Kafka comparison |

Two consequences. Loops 0–2 compress to roughly 1.5 weeks for you — but do not skip Loop 0's geometry or Loop 2's noise/hysteresis work, because those are the parts that make the simulation *credible to a virtual-fencing engineer*, and nothing in OPD taught them. And carry your OPD habits over on day one: Makefile, `make up`/`make test`, ADRs, race detector in CI, break-it-first.

---

## Loop 0 — One cow, one terminal (2–4 days)

**Goal:** prove you can model the domain in plain Go before any infrastructure exists.

Build a single binary: one simulated cow doing a **correlated random walk** (mostly keeps its heading, occasionally turns, occasionally stops to graze) around a real lat/lng point — pick a real farm-sized area on Google Maps near you. Print its position every second. Add a hardcoded rectangular fence and print `WARNING` when the cow is within 10 m of the edge, `BREACH` when outside.

**Learn:** the haversine formula (write it yourself once before using a library — 20 lines, and you'll never be confused by "distance between two GPS points" again); degrees-vs-meters, *properly*: one degree of latitude is ~111 km everywhere, but one degree of longitude shrinks by cos(latitude) — at Ludhiana (~31°N) it's ~95 km, at Halter's Waikato farms (~37°S) ~89 km. Move cows in meters and convert with that correction, or every herd on your map drifts stretched east–west and a fencing engineer spots it in ten seconds. Also from day one: a **seeded random source** (`rand.New(rand.NewSource(seed))`, never the global one) and an **injectable clock** (`type Clock interface { Now() time.Time }`, like your OPD project) so the same seed always produces the same cow path — the whole future test suite depends on this being deterministic.

If OPD didn't give you goroutine confidence, do the **concurrency preview** before leaving: three cows as three goroutines with a `sync.WaitGroup`, each printing with its cow ID. If it did, skip it.

**Definition of Done:** `go run .` shows a believable cow wandering and occasionally breaching; `haversine_test.go` passes table-driven tests against known city-pair distances; a **golden test** proves seed 42 produces the identical path twice.

**Explain test:** Why can't you compare raw lat/lng deltas to a meter threshold? Why is a longitude degree shorter than a latitude degree? What makes a random walk "correlated," and why does it look more realistic?

---

## Loop 1 — The walking skeleton (1 week)

**Goal:** the entire system in one binary — every future service exists here as a Go package.

One process, three internal parts communicating over **channels**: a `simulator` package running N cows as goroutines, a `collar` package holding each cow's fence and running the escalation state machine (`inside → warning → breached → escaped`, with audio/vibration/pulse as logged "cues"), and a `server` package exposing REST endpoints: `GET /cows` (live positions), `GET /alerts`, `PUT /fence` (update the polygon — all collars pick it up). Storage is in-memory maps guarded properly.

Build it in this order, keeping it runnable at every step: **(a)** N cows in goroutines all sending positions into one channel, with a single consumer printing them — that's the fan-in; **(b)** add the collar state machine inside that consumer path; **(c)** put the HTTP server on top and wire graceful shutdown across all three parts last.

**Make the cow a cow, not a particle.** Three behaviors turn a random walk into a credible animal, and all three are cheap in Loop 1 and expensive to retrofit later:

- **Cue response.** This is the entire premise of virtual fencing and it was missing from the model: when the collar issues an audio cue, the cow turns away from the boundary with probability *p* (say 0.85); on the pulse, 0.98. A small fraction of cows are "untrained" with lower *p*. Without this, cows never "walk back in" after a fence change — they random-walk regardless and breaches resolve by luck, and any fencing engineer watching your demo will notice. With it, you get the metric Halter's Guidance team actually lives on: **cue compliance rate** — audio cues that resolved without escalating to a pulse.
- **Herd cohesion.** Cows are herd animals: add a weak pull toward the herd centroid (boids-lite: cohesion + the existing heading persistence). Without it, every cow is "far from the herd" all the time and Loop 5's herd-outlier detector is meaningless.
- **Activity modes.** The correlated walk already has modes — make them explicit states (grazing: slow, wandering; walking: fast, directed; resting: stationary) with realistic dwell times and report the state in each ping. Their collars carry IMU sensors precisely to classify this; your simulator emitting it from Loop 1 makes the map richer for free.

**The time model — decide it now, in an ADR, because it silently breaks Loop 4 otherwise.** Give the simulation a `--speed` knob, but be precise about what it scales: **speed scales the physics, never the clock.** `--speed=60` means each real-time tick advances the cow's *movement* by 60 simulated seconds; every timestamp the collar reports is still real wall-clock time. If you instead fast-forward the timestamps, Loop 4's "reject pings from the future" validation will reject every ping your simulator sends, and every wall-clock threshold in the backend (stale-collar minutes, cooldowns, retention) becomes meaningless. Consequences: the collar's hysteresis dwell time is counted in *ticks*, backend thresholds stay in wall-clock, and a "simulated grazing day" runs in 24 real minutes at speed 60.

**Learn:** the new pattern here is *many-producer fan-in* — N goroutines into one channel with a consumer that must never fall behind — which is different from OPD's worker-pool and SSE-broker shapes; `sync.RWMutex` vs channels and when each fits; simulation-clock design and why production code must never call `time.Now()` directly; the difference between simulated physics time and wall time. `context`, graceful shutdown, `net/http`, and `slog` you already own from OPD — reuse your patterns, don't relearn them.

**Definition of Done:** 50 cows run concurrently; `curl` shows live positions; updating the fence via the API changes collar behavior within one tick *and cows visibly turn back on cues*; the herd stays loosely clustered; `go test -race ./...` passes; shutdown is clean with no goroutine leaks; the same seed at `--speed=60` reproduces the same breach sequence twice; the time-model ADR exists.

**Explain test:** Walk through what happens to every goroutine when SIGINT arrives. What happens to the fan-in channel if the consumer is slower than the producers, and what are your three options? What does `--speed` scale and what does it deliberately not scale, and why? Where would this design break at 100,000 cows?

**Read alongside (only if concurrency still feels shaky):** *The Go Programming Language* (Donovan & Kernighan) chapters on goroutines/channels, or the free "Learn Go with Tests" site.

---

## Loop 2 — Two services, real geometry, real storage (1.5–2 weeks)

**Goal:** split the monolith along the seam that matters — device side vs cloud side — and replace toy geometry with the real thing.

Split into two deployables: **`collarsim`** (cows + collars; this simulates the *fleet of devices*) and **`backend`** (ingest, fence management, alerts). For now they talk plain HTTP: collars POST pings, poll for fence updates. It will feel clunky — that's deliberate; Loop 3 fixes it and you'll understand *why* MQTT exists.

Replace rectangles with **arbitrary polygons** using `paulmach/orb`: point-in-polygon for containment, distance-to-boundary for the warning zone. Fences are GeoJSON. Add **PostgreSQL** via `pgx`: tables for collars, fences (store GeoJSON for now — PostGIS comes in Loop 5), pings, and alerts. Put a `farm_id` column on every table and scope every query by it from the first migration — multi-tenancy costs nothing today, mirrors how Halter actually serves thousands of farms, and Loop 5 hardens it into a real wall. Add schema migrations (`goose` or `atlas`). Wrap everything in **Docker Compose**.

Also give the project a face: a single static HTML page served by the backend with a read-only **Leaflet** map — the fence polygon plus moving cow dots, polling `GET /cows` once a second. Roughly 60 lines of vanilla JS, and from this loop onward every demo happens in a browser on a real map, not in a terminal. (If you want something public early, this version deploys fine to a $5 VPS with Compose — but the proper cloud deployment is Loop 9; don't burn time on it yet.)

Now make the GPS honest, because this is where a fencing engineer decides whether your simulation is real. Real GPS fixes carry **noise** — add Gaussian jitter of a few meters (σ ≈ 3–5 m, occasionally a 20 m outlier) to every reported position. Watch what happens: a cow grazing near the boundary now *flaps* — inside, outside, inside — and your state machine fires a breach alert every other tick. That pain is the lesson. Cure it with **hysteresis and dwell time** in the collar state machine: the warning zone starts N meters *inside* the boundary, a breach requires being more than M meters outside for at least T seconds, and returning to "inside" requires being clearly inside, not just on the line. This is exactly how real collars avoid shocking a cow for a GPS wobble, and it's the first thing Halter's Guidance team would ask you about.

**Learn:** service decomposition and API contracts; writing an HTTP *client* properly on the collar side (timeouts, retries — your first taste of "the network is unreliable"); polygon geometry with `orb` and GeoJSON; noise modeling and hysteresis/debounce as a general technique (it reappears in alerting thresholds, autoscaling, and every sensor pipeline); Dockerfiles with multi-stage builds. `pgx`, migrations, SQL, Compose, and CI skeletons you own from OPD — reuse them (`go vet` + `go test -race` on every push from this loop on). For the map, start from the Leaflet quickstart — copy-and-adapt, not a JS course.

**Definition of Done:** `docker compose up` starts postgres + backend + collarsim; you can PUT a weird L-shaped paddock as GeoJSON and watch cows respect it *on the map in your browser*; pings and alerts survive a backend restart; with noise on, a cow parked on the boundary for a simulated hour produces **at most one** breach alert, not sixty.

**Explain test:** Why does fence *enforcement* live in the collar process while fence *authoring* lives in the backend? (This edge-vs-cloud split is the most important architectural fact about real virtual fencing — collars can't wait on a network round-trip to decide whether to beep.) Why is a single threshold the wrong design for any noisy sensor? What breaks if two backend instances write alerts simultaneously?

---

## Loop 3 — Real IoT transport: MQTT (1.5–2 weeks)

**Goal:** make device communication work the way internet-connected collars actually work.

Replace HTTP polling with **MQTT**. Run NATS with its MQTT gateway enabled (one container gives you both MQTT broker and, next loop, JetStream; note it speaks MQTT 3.1.1 with QoS 0 and 1 only — fine for this system, use EMQX if you ever want to experiment with QoS 2) — or EMQX if you want a dedicated broker with a nice dashboard. Design the topic tree deliberately, and **farm-scope it from the first message**, because Loop 6's per-collar ACLs are only expressible if the farm is in the topic path:

- `farm/{farm_id}/collar/{id}/ping` — position telemetry (QoS 0: losing one ping is fine)
- `farm/{farm_id}/collar/{id}/event` — state transitions like breach (QoS 1: must arrive)
- `farm/{farm_id}/fence/{paddock_id}` — **retained message** downlink: a collar that connects late immediately receives the current fence, exactly like a real collar syncing after being out of coverage. Deleting a paddock must publish an *empty* retained message to clear it, or ghost fences haunt every reconnect — a classic MQTT gotcha
- `farm/{farm_id}/collar/{id}/assignment` — retained: which paddock this collar belongs to. Without it a collar can't know which fence topic to subscribe to; assignment is state that must be pushed, same as the fence
- **Last Will & Testament** on each collar connection → broker publishes `farm/{farm_id}/collar/{id}/status` = `disconnected` if a collar dies without saying goodbye. The backend records *status and since-when*; it does **not** alert on it directly. Brief disconnects are normal for field devices (Loop 4 will make them constant), so "collar dark" is a *derived* condition — disconnected for longer than a threshold, evaluated by the stale-collar check in Loop 5. Same hysteresis lesson as GPS noise, applied to connectivity

**Learn:** MQTT concepts that transfer to every IoT job: topics and wildcards, QoS 0/1/2 and what "at least once" really costs, retained messages, LWT, keep-alive; the Eclipse Paho Go client; designing payloads (start JSON, note where protobuf would shrink them).

**Definition of Done:** kill a single collar goroutine → the backend marks it `disconnected` within seconds (the alert threshold is a policy, applied in Loop 5); update a fence while half the collars are "offline" → they receive it the moment they reconnect, from the retained message, with zero backend code for catch-up; delete a paddock → reconnecting collars do *not* receive the stale fence.

**Explain test:** Why QoS 0 for pings but QoS 1 for breach events? What exactly does the broker do with a retained message, and how do you un-retain one? Why record LWT as status rather than alerting on it? How does LWT differ from the backend noticing missing pings, and why do you eventually want both?

---

## Loop 4 — Durability and the event-driven core (2 weeks)

**Goal:** make the backend survive crashes, floods, and lying networks. This is the loop that separates portfolio projects from toys.

Insert **NATS JetStream** between ingest and processing. The ingest service does one job: validate MQTT messages and publish them to a durable stream. "Validate" means real sanity checks, not just JSON parsing: reject or flag pings timestamped in the future, older than the buffer window, or implying impossible movement (a cow that "moved" 2 km in one second is a bad GPS fix or a spoofed device — drop it and count it). Independent consumers do the rest: a **geofence processor** (server-side breach checking as defense-in-depth), an **alerter** (for now it only writes alert records — human delivery like Telegram arrives in Loop 6), and a **persister** writing to Postgres. Each is a separate Go service with its own durable consumer — the same decomposition muscle you built in Loop 2, flexed three more times. This loop also changes the collar payload (sequence numbers, below, plus a `fence_version` so the backend knows *which* fence the collar was enforcing when it reported — without that, server-side and collar-side verdicts can't be compared honestly), so version your schema now (`"v": 2` in every message): real device fleets never all upgrade at once, and handling two payload versions side by side is an extremely realistic detail.

Now make the simulator hostile, because real rural connectivity is: give each simulated collar random **connectivity dropouts**, during which it *buffers* pings locally and replays them on reconnect — timestamped in the past, out of order, and sometimes duplicated. Your pipeline must handle all three:

- **Idempotency:** every message carries `(collar_id, monotonic_seq)`; consumers dedupe.
- **Out-of-order:** persist by *event time*, not arrival time; breach logic must not "un-breach" a cow because a stale ping arrived late.
- **Backpressure:** slow the persister artificially and watch consumer lag grow; understand acks, redelivery, and max-in-flight before fixing it.

**Now the deepest design decision in the whole system, which this loop forces:** you now have *two* things that can say "breach" — the collar (edge verdict, sent as an event) and the server-side geofence processor (cloud verdict, computed from pings). If both can create alerts, every breach notifies twice, and the Loop 6 dedupe key has nothing sound to key on. Resolve it explicitly:

- The **collar mints a breach episode** — `episode_id = (collar_id, episode_seq)` — when it enters `breached`, and closes it when it returns to `inside`. Every event carries its episode. An *alert* is keyed by episode, so retries, replays, and duplicates of the same episode collapse to one alert by construction.
- The **server-side processor never creates alerts**. It corroborates: for each ping it computes its own inside/outside verdict against the fence version the collar reported, and when its verdict disagrees with the collar's state for longer than a tolerance, it emits a `verdict_mismatch` event — the signal that a collar's firmware, fence copy, or GPS is wrong. Halter's ML and R&D teams live on exactly this signal.
- Write the ADR: edge is authoritative for *action* (cues fire on the collar, latency matters), cloud is authoritative for *history and audit*. Two authorities for the same decision is the classic distributed-systems bug; you're choosing not to have it.

**Learn:** streams vs subjects, durable consumers, ack/nak/term semantics, redelivery and poison messages, at-least-once vs exactly-once (and why "exactly once" is mostly idempotency wearing a suit); event time vs processing time — the core distinction of all streaming systems. Stay on JSON here *deliberately* — readable payloads make this loop's debugging vastly easier — but know that real fleets don't: Halter ships nanopb (protobuf in C) on the collar, and Loop 10 converts your payload and measures what it saves. Write that trade-off in the loop's ADR now, so the choice reads as staged rather than naive.

**Definition of Done:** `docker compose kill` the persister mid-flood, restart it, and prove zero pings lost and zero alerts duplicated (write an integration test that does exactly this — a Go test or script against your running Compose stack is fine for now, Loop 7 formalizes it with testcontainers — it's your best interview story); a chaos flag in the simulator toggles dropout behavior; a replayed, duplicated breach episode produces exactly one alert row; a deliberately corrupted fence copy on one collar produces a `verdict_mismatch` within a minute.

**Explain test:** Where can a message be lost in collar → MQTT → JetStream → consumer → Postgres, and what guards each hop? Why did you put dedup in the consumer rather than the ingest? Why does the collar, not the server, mint the breach episode — and what would go wrong the other way round?

**Read alongside:** *Designing Data-Intensive Applications* (Kleppmann) — start it this loop, read it slowly across the rest of the project. It is the book behind every senior backend interview.

---

## Loop 5 — Geospatial and time-series depth (2–3 weeks)

**Goal:** move from "geometry in app code" to "geometry as a database superpower," and make the data layer scale.

This loop carries six pieces, so build them in this order, each one runnable before the next: **(a)** PostGIS geometry columns + `ST_Contains` on fences; **(b)** Timescale hypertable for pings + retention; **(c)** the continuous aggregate; **(d)** server-side detections and escalation; **(e)** RLS; **(f)** backup + restore drill. If time is short, (c) is the only item that can slide into Loop 10.

Enable **PostGIS**: fences become `GEOMETRY(POLYGON, 4326)` columns with GiST indexes; breach checking becomes `ST_Contains` / `ST_DWithin` queries; you can now answer "which cows are inside paddock 7 right now" in one indexed query. Enable **TimescaleDB**: pings become a hypertable with automatic time partitioning; add retention policies (drop raw pings after 30 days) and a continuous aggregate (per-cow daily distance traveled — that's a grazing-behavior feature, the kind of thing Halter's ML runs on).

Add the smarter server-side detections that only the cloud can do (the collar only knows about itself): **herd centroid** and "cow suspiciously far from herd" (meaningful only because Loop 1 gave cows cohesion — a herd of independent random walkers has no herd), **collar-dark detection** — the policy layer over Loop 3's LWT status: `disconnected` or no ping for longer than N minutes raises the alert, brief blips don't — and a **breach escalation flow**: if an episode stays open for more than X minutes, escalate the alert severity. To get the 1M+ historical pings this loop's DoD needs, run the simulator at high `--speed` with a `--backfill` flag that writes past wall-clock timestamps directly — the one place fabricated timestamps are allowed, and only because they're explicitly historical.

Two hardening items transplanted from your OPD plan, because they belong in any multi-farm system. First, make `farm_id` a real wall: seed a second farm, add Postgres **row-level security** (`SET LOCAL app.farm_id` per transaction, policy checks it), and write the test that *deliberately drops the `farm_id` filter* and still can't read the other farm's cows — because the database itself refuses. Second, **nightly `pg_dump`** to object storage and one real **restore drill**: drop the database, restore it, prove nothing was lost. Most portfolios skip the restore; that's exactly why you won't.

**Learn:** spatial types, SRIDs (4326 = GPS), GiST indexes and `EXPLAIN ANALYZE` on spatial queries; hypertables, chunks, continuous aggregates, retention; when to compute in SQL vs in Go.

**Definition of Done:** 1M+ synthetic historical pings loaded; "cows in paddock now" and "this cow's path yesterday" both return in milliseconds with `EXPLAIN` output you can read; escalation flow demonstrably fires.

**Explain test:** What does a GiST index actually index? Why does Timescale chunk by time? When is it wrong to push logic into the database?

---

## Loop 6 — The owner-facing product (3 weeks)

**Goal:** someone who is not you can use this.

This is the heaviest loop in the roadmap. Build order, each step shippable: **(a)** owner auth + API hardening; **(b)** transactional-outbox alert delivery + Telegram; **(c)** Valkey-backed WebSocket feed; **(d)** React console with fence drawing; **(e)** optimistic locking on fences; **(f)** per-collar credentials + topic ACLs. Coming from OPD, (a), (b), and (e) are patterns you've already shipped — budget most of the three weeks for (c), (d), and (f).

Harden the API: authentication (owner accounts with JWT or PASETO sessions; the WebSocket authenticates the same way — token on connect, farm-scoped subscriptions, never an open socket), request validation, rate limiting, consistent error shapes. Give every collar its own **device credential** for MQTT — real fleets never share one password across devices — and pair it with **topic ACLs** on the broker: collar `abc` on farm `f1` may publish only to `farm/f1/collar/abc/#` and subscribe only to `farm/f1/fence/#` and its own `assignment`. Credentials without ACLs are theater: a single compromised collar could still publish fake positions for every other cow on the farm. With ACLs, the blast radius of one stolen credential is one cow — and that sentence is your Loop 9 Explain-test answer. Add a **WebSocket** endpoint streaming live positions and alerts, and back it with **Valkey** (Redis-compatible, so any Redis client and all your Redis knowledge apply — `redis-go` talks to it unchanged): latest position per cow in a hash, fan-out via pub/sub, while Postgres stays the system of record. The pub/sub isn't decoration: the moment Loop 8 runs two API replicas, a browser connected to replica A must still see updates that arrived at replica B, and Valkey is the bus that makes that true. That's the honest way to cover the "relational *and* non-relational databases" line in every Halter JD — a hot cache doing a job Postgres shouldn't, not padding.

Build **alert delivery** as a **transactional outbox** — the crown jewel of your OPD roadmap, transplanted whole. The alert record and its outbox row are written in the *same transaction*; a worker claims rows with `SELECT ... FOR UPDATE SKIP LOCKED`, sends to Telegram (easiest to demo) or a webhook, and marks them sent — with jittered exponential backoff, max attempts → dead-letter status, a `dedupe_key` unique index (the Loop 4 **episode_id** plus severity — this is exactly why episodes exist) so a retried breach can never notify twice, and per-cow **cooldown** so one confused cow doesn't send 400 messages. Test it against a *hostile fake* notifier that injects timeout-after-success (the classic duplicate cause) and 500s, while you kill the worker mid-broadcast — the fake counts messages by dedupe key and the count must be exact. And add a `version` column on fences: two owners editing the same paddock → the second save gets "fence changed, reload" via `UPDATE ... WHERE version = $1` — optimistic locking learned in one afternoon.

Then upgrade the Loop 2 map page into the **owner console** — this is the loop where the project starts looking like Halter's product instead of a dev tool:

- **Draw fences on the map.** Add the Leaflet-Geoman (or Leaflet.draw) plugin so the owner draws, reshapes, and deletes paddock polygons directly on the map. Saving fires your fence API → backend publishes the retained MQTT downlink → collars adopt the new boundary. The end-to-end demo: drag a fence edge inward, and within seconds cows that are now "outside" start getting cues and walking back in. This map-to-collar round trip is the single most convincing thing in your demo video.
- **Live movement, no polling.** Cow dots update over the WebSocket feed, each with a fading trail of its last few minutes; warning-state cows turn amber, breaches flash red.
- **Alert feed + cow detail.** A side panel streaming alerts as they happen, and click-a-cow for its status, battery, and trail history.

Build the console in **React** — for most people I'd say vanilla JS to protect their time, but you already have frontend experience and React is Halter's web stack, so this converts your past skills into "ships across their exact stack" instead of leaving them off the table. Still keep it one page and skip the design-system rabbit hole; the value is the live system behind it. Do borrow their mobile team's obsessions in miniature: handle WebSocket disconnects gracefully, show a stale-data indicator when the feed drops, reconnect with backoff — offline-first, field-conditions thinking is a named hiring signal for them.

**Learn:** authN vs authZ, token design, device identity for IoT fleets; WebSocket lifecycle in Go (write pumps, ping/pong, slow-client handling); Valkey/Redis data structures and pub/sub, and the cache-vs-source-of-truth discipline; delivery guarantees for notifications; API design taste; React + Leaflet integration and reconnect/stale-state UX.

**Definition of Done:** a stranger with the README can register, *draw a fence on the map with their mouse*, watch cows react to it live, and receive a Telegram message with a Google Maps link when one escapes.

**Explain test:** What happens to your WebSocket handler when a client on a slow connection stops reading? How does per-device authentication change what a leaked credential can do?

---

## Loop 7 — Observability and proof of reliability (2 weeks)

**Goal:** stop believing the system works; measure it. "Reliable" is a claim about evidence.

Wire **Prometheus** metrics into every service: pings/sec, JetStream consumer lag, WebSocket client count, DB pool saturation, Valkey hit rate on the live-position cache, cue compliance rate, verdict-mismatch rate — and the headline: **breach-to-notification latency**, as *two* histograms, because one number is a trap. *Pipeline latency* runs from the moment the breach event hits ingest to the moment the outbox marks the notification sent; that is the SLO you own and can defend (say p99 < 2 s). *End-to-end latency* runs from the collar's event timestamp to notification and includes however long the collar sat in a connectivity dropout buffering — it's unbounded by design, so you report it, you don't promise it. Interviewers will try to collapse these into one; don't let them. Build a **Grafana** dashboard. Add **OpenTelemetry tracing** so one breach episode can be followed collar → ingest → stream → processor → outbox → Telegram with a single trace ID; propagate that ID into every `slog` line. Add alerting rules (consumer lag too high, no pings for 5 min, mismatch rate above baseline).

Two additions here that the next loops depend on. First, wire `/healthz` (process alive) and `/readyz` (NATS, DB, and Valkey connections verified) endpoints into every service — Loop 8's Kubernetes probes will point at exactly these. Second, this is where **gRPC** enters: give `collarsim` a gRPC **control plane** — `SpawnCows(n)`, `SetDropoutRate(x)`, `TeleportCow(id, lat, lng)` — and drive it from a small CLI. `TeleportCow` doubles as your demo button: force a breach on cue during the video instead of waiting for a cow to wander out.

Then attack it: use the control plane as your **load generator** to run 10,000+ cows (or a k6 script against ingest). Expect the first wall to be the operating system, not your code: 10k MQTT connections from one process means 10k file descriptors and 10k ephemeral ports — raise `ulimit -n`, check the broker's connection limits, and consider multiplexing several simulated collars per connection *only after* measuring the honest one-connection-per-collar number. Then find the first real bottleneck with **pprof** (CPU + heap) and fix it; chaos-test by killing the broker mid-run and verifying recovery. Round out the test pyramid: table-driven unit tests, **testcontainers-go** integration tests (real Postgres + NATS in tests), race detector in CI.

**Learn:** the metrics/logs/traces triad and what each is actually for; histogram buckets and percentiles (why p99 ≠ average); protobuf service definitions and gRPC codegen; pprof and Go performance intuition (allocation pressure, GC); load-testing methodology; upgrading your Loop 2 CI skeleton into a full pipeline (golangci-lint → test → build).

**Definition of Done:** a dashboard screenshot showing the system holding N-thousand cows with p99 breach-latency under a target *you chose and can defend*; a `LOAD.md` writing up the bottleneck you found and the fix, with before/after numbers.

**Explain test:** What's your breach-detection SLO and why? Show the trace of one alert. What did pprof reveal?

---

## Loop 8 — Kubernetes and failure-proofing (2 weeks)

**Goal:** run the whole system the way the industry runs systems, and prove it survives what Kubernetes does to processes.

Move Compose to a local **kind** or **k3d** cluster: Deployments for each service, StatefulSet (or the NATS Helm chart) for JetStream, Valkey via a simple Deployment or Helm chart (their own `valkey-cluster-operator` is the production-grade route — worth reading its Go source even if you deploy the simple way), liveness/readiness probes pointed at the `/healthz` and `/readyz` endpoints you built in Loop 7 (readiness = "connected to NATS, DB, and Valkey," not just "process up"), resource requests/limits, HPA on the geofence processor, ConfigMaps/Secrets. One trap to know in advance: **MQTT is raw TCP, not HTTP** — it cannot go through an HTTP Ingress. Expose the broker with a `LoadBalancer`/`NodePort` Service or a TCP route in your ingress controller (Traefik `IngressRouteTCP`, or the nginx TCP configmap), and keep the HTTP API and WebSocket on the normal Ingress. Nearly everyone loses an afternoon here; now you won't. Your Loop 1 graceful-shutdown work pays off here: rolling deploys must lose zero messages, and now you can prove it.

Failure drills, each written up in `NOTES.md`: kill the processor pod mid-flood (verify redelivery), kill a NATS pod (verify stream survival with replicas), roll a new backend version during load (verify zero dropped WebSocket alerts or explain the reconnect story).

**The trap inside "HPA on the geofence processor":** the moment two processor replicas pull from the same consumer, two pings from the *same collar* can be processed concurrently and out of order — and you've reintroduced the exact bug Loop 4 fixed, one layer up. You have two honest choices, and you must pick one in an ADR: **(a)** keep the processor order-tolerant — every verdict compares event time against the collar's last-seen event time in Valkey, so a stale ping processed late can never move state backwards, and accept that "ordered per collar" is a property you *don't* rely on; or **(b)** partition — hash `collar_id` into N subjects, one durable consumer per partition, so exactly one worker owns a collar at a time, and scaling means re-partitioning. (a) is simpler and what most telemetry systems do; (b) is what Kafka people expect. Knowing why you chose is the interview answer; scaling without choosing is the outage.

**Learn:** core K8s objects and reconciliation thinking; probes, PodDisruptionBudgets, graceful termination (`SIGTERM` → drain → exit before the grace period); Helm basics; reading `kubectl describe` under stress.

**Definition of Done:** `make cluster-up` brings up everything on a fresh machine; all three failure drills pass with evidence.

**Explain test:** Difference between liveness and readiness, and a concrete way a wrong liveness probe causes an outage? What exactly happens between SIGTERM and SIGKILL in your services? What breaks when you scale a stream consumer from one replica to two, and which of the two fixes did you choose?

---

## Loop 9 — Cloud, IaC, CI/CD, security (2 weeks)

**Goal:** it runs on the public internet, deployed by robots, and you can say "AWS and Terraform at scale" with a straight face.

**Terraform** an AWS deployment. Budget-honest path: **k3s on a single EC2 instance** — a one-binary Kubernetes that runs your Loop 8 manifests essentially unchanged, so nothing from last loop is thrown away (a small managed EKS cluster if you can afford ~a month of it; plain Compose on a VPS only as a last-resort fallback) — plus RDS Postgres or Postgres on the instance, and your NATS. Terraform the VPC, security groups, instances, DNS. Extend GitHub Actions to full CD: on merge → build → push image → deploy. Secrets via SSM/SOPS, never in the repo.

Now upgrade device identity to **mTLS with per-collar X.509 certificates**, promoting the credentials from Loop 6. This isn't gold-plating: Halter's firmware forks (Amazon FreeRTOS on Espressif) point straight at AWS IoT Core, whose native device auth is exactly per-device X.509 mutual TLS with policy-scoped topics. Build your own tiny CA, issue a cert per collar, have the broker verify the chain and map the certificate CN to the collar's topic ACLs, and write a **revocation** path — one stolen collar must be revocable without touching the other 999. Almost no portfolio project has device PKI; this one does, and it maps 1:1 to how their fleet actually authenticates. If you have budget and curiosity, stand up a handful of collars against real **AWS IoT Core** for a week and write the comparison (managed certs, policies, and rules-engine routing vs your self-hosted broker) — that comparison is a genuinely senior conversation.

Then run `collarsim` from your own laptop against the cloud backend: your "devices" are now genuinely remote, on a genuinely unreliable network — the internet-not-LoRa constraint you set at the start is now the production topology.

Also move two pieces onto Halter's own AWS nouns (their data stack runs on Athena, S3, Kinesis, SQS and SNS): route alert fan-out through **SNS → SQS**, replacing part of the hand-rolled delivery path you built in Loop 6 — building it by hand first, then swapping in the managed service, is the spiral working as intended. One catch that separates people who've used SQS from people who've read about it: standard SQS is **at-least-once**, so the swap silently reintroduces the duplicate-notification bug you killed in Loop 6 unless you either use an SQS **FIFO** queue with `MessageDeduplicationId = episode_id` or keep the Telegram sender idempotent on the dedupe key. Run the Loop 6 hostile-fake test against the new path and make it pass again — that's the point of having it. And archive raw pings to **S3** with lifecycle rules (queryable later via Athena if you're curious). Your resume now shares five real nouns with their production infrastructure.

**Learn:** Terraform state, plan/apply discipline, modules; VPC/SG basics; TLS and cert issuance (Let's Encrypt / cert-manager); CD pipelines and rollback; queue-vs-stream trade-offs from swapping a hand-rolled path for managed services; cloud cost awareness (tear it down with `terraform destroy` between sessions — the skill is the code, not the uptime bill).

**Definition of Done:** fresh clone → `terraform apply` + one pipeline run → live system on a real domain, collars authenticating over TLS from your laptop; teardown is one command. The payoff: you send anyone a link, they open the map on their phone and watch your simulated herd grazing in real time — and you redraw a fence from your own phone and they see the cows react. That link goes in the email, your resume, and your LinkedIn.

**Explain test:** Where is your Terraform state and why does it matter? What can an attacker do with one stolen collar credential, and what's your blast radius?

---

## Loop 10 — Breadth, polish, and the email (1–2 weeks)

**Goal:** convert the system into career artifacts.

- **Kafka literacy:** add Kafka (Redpanda locally is painless) as an *additional* sink — bridge breach events from JetStream into a Kafka topic with a small Go consumer on the far side. Now you can compare partitions/consumer-groups vs streams/durable-consumers from experience, which is exactly how staff engineers talk about tools.
- **Protobuf (do this one).** Convert the collar payload from JSON to protobuf and benchmark both — bytes on the wire and encode/decode time. Their GitHub carries forks of **nanopb** (protobuf for constrained C devices) and **ts-protoc-gen** (protobuf → TypeScript), so protobuf is their format from collar to cloud to browser. You already generate Go from `.proto` for the Loop 7 gRPC control plane, so the toolchain is in place. Say the number out loud in interviews: "JSON ping was N bytes, protobuf was M — at a million collars pinging every minute that's X GB/day of radio time saved."
- **Domain flourishes** (pick one or two): a battery + solar model with low-battery alerts; scheduled "shifts" that move the fence polygon at 6 a.m. like Halter's remote herding — the feature their marketing leads with; a per-farm **cue-compliance report** (the Loop 1 metric, aggregated over Loop 5's continuous aggregates) — the exact artifact their Guidance data analysts produce.
- **Stretch goal — write a tiny Kubernetes operator in Go.** Define a `Farm` CRD (herd size, paddock geometry, dropout rate) and a controller that reconciles it into a running `collarsim` Deployment. It's a weekend with kubebuilder, it teaches the reconcile loop from the inside, and it directly mirrors the **valkey-cluster-operator** they build and maintain in Go. If you want one thing on your resume that makes their Platform team look twice, this is it.
- **Farm field study:** visit one or two dairy farms near you and ask how they actually track, contain, and move animals — then write half a page on what your simulation got wrong and what you changed. "Get out on farm" is written into nearly every Halter JD; doing it before applying, from one of the world's biggest dairy regions, is a signal no other applicant will have.
- **The artifacts:** an architecture diagram + honest README (including "what I'd do differently"); a 2–3 minute demo video (map, live cows, a breach, the Telegram ping, the Grafana dashboard); one deep-dive blog post from your best `NOTES.md` story — the crash-recovery test or the pprof hunt are the strongest candidates.
- **The email:** short. "I spent four months building a simulation of a collar-to-cloud virtual fencing pipeline to understand your problem space — edge-side fence enforcement, MQTT with buffered replay, JetStream, PostGIS, K8s — a small version of your half-billion-messages-a-day problem. Demo video and writeup here. I'd love to talk about the real thing." Frame it as studying *their* problem, not cloning their product, and name the teams your project maps to (Guidance, Platform Communications, Platform). Apply to the evergreen **Junior/Intermediate Engineers – Product & Engineering** listing with the cover letter they explicitly ask for, and separately send the live link to their engineers and engineering managers on LinkedIn — a working demo gets forwarded internally, and that's the fast path. It also works as a flagship project for any Go/infra role, not only Halter.

**Explain test:** When would you choose Kafka over JetStream for this system, and what would it cost you operationally? What did you learn on a real farm that your simulation had wrong?

---

## If you hit a "gap" — the self-repair protocol

Somewhere in month two or three, something will feel like a hole in this plan. It almost certainly isn't. Diagnose it against these four cases and apply the fix — you don't need anyone's re-verification:

1. **"I don't know X and this step needs it."** Not a roadmap hole — no roadmap can list every micro-skill, and learning-on-demand *is* the spiral. Rule: spend up to one day on the official docs or tutorial for X, build the smallest possible throwaway example of X alone, then return. If X routinely takes people weeks (it won't here — every big topic already has a loop), only then is it a real gap.
2. **"This step assumes something my system doesn't have."** Check the handoff chain section. In practice this always means a Definition of Done was skipped or half-done in an earlier loop. Go back, finish that DoD, return. The DoD gates *are* the gap-prevention mechanism — they only work if you don't wave yourself through them.
3. **"The tool doesn't work like the roadmap says."** Expected: over months, libraries and versions drift. The *architecture* (edge enforcement, MQTT semantics, durable stream, spatial DB, K8s, IaC) is the stable part of this document; any named library is replaceable. Trust the tool's current official docs over this file, swap equivalents freely (EMQX↔NATS-MQTT, kind↔k3d, chart↔manifest), and note the swap in `NOTES.md`.
4. **"I've been stuck on the same thing for days."** Timebox rule: after two days stuck, shrink scope until it works — 5 cows instead of 500, one consumer instead of three, skip the optional flourish — get the loop's DoD passing at the smaller scale, write down what beat you, and move on. Half the entries in a senior engineer's story bank start exactly this way.

One meta-rule: this file is a map, not a contract. If you deviate for a good reason you can write down in one sentence, the deviation is correct.

---

## The handoff chain — proof there are no gaps

Every transition works the same way: loop N leaves behind exactly the pieces loop N+1 builds on, and ends with a *pain* that loop N+1 cures. If you ever feel a jump, check this chain — you probably skipped a Definition of Done.

- **0 → 1:** hands over the cow model, correct geometry (haversine, cos-latitude), a seeded RNG and injectable clock. Pain that motivates 1: one cow in one terminal isn't a system.
- **1 → 2:** hands over fan-in, the state machine, a cow that responds to cues and stays with its herd, the physics-only `--speed` model (ADR), graceful shutdown, and a working API. Pain: everything vanishes on restart, and one process pretends to be both device and cloud.
- **2 → 3:** hands over the device/cloud split, API contracts, Postgres, Docker, the map, and a collar state machine that survives GPS noise via hysteresis. Pain: collars polling HTTP for fence updates is laggy, wasteful, and nothing detects a dead collar.
- **3 → 4:** hands over farm-scoped MQTT topics, retained fence + assignment downlinks, and LWT recorded as status. Pain: if the backend restarts, in-flight messages hit the floor — nothing durable sits between broker and database.
- **4 → 5:** hands over the durable pipeline, split consumers, the breach-episode model with one alert authority, verdict-mismatch signals, and a persister filling tables with pings. Pain: geometry in app code and unindexed history queries get slow and clumsy as data grows.
- **5 → 6:** hands over fast spatial/time-series queries, multi-farm tenancy hardened with RLS, backups with a proven restore, escalation, and richer detections. Pain: a powerful system nobody but you can log into, see, or get alerts from.
- **6 → 7:** hands over a complete product — auth, React console with live map and fence drawing, Valkey-backed feed, outbox-driven Telegram delivery, optimistic locking on fences, per-collar credentials with topic ACLs. Pain: you *believe* it's reliable but can't measure, prove, or stress any of it.
- **7 → 8:** hands over two honest latency histograms with a defended SLO, traces, health endpoints ready for probes, and a gRPC-driven load generator. Pain: Compose can't do rolling deploys, self-healing, or autoscaling, so your reliability story caps out.
- **8 → 9:** hands over battle-tested Kubernetes manifests (services, NATS, Valkey), failure drills, and a written scaling-vs-ordering decision. Pain: it all still only exists on your laptop — nobody can open the link.
- **9 → 10:** hands over a live public system with CD and per-collar mTLS device identity. No pain left to cure — Loop 10 just harvests the career value: protobuf, Kafka breadth, the operator stretch, artifacts, the email.

---

## Threads that run through every loop

- **Git discipline:** meaningful commits, PR-style descriptions even solo. The history is part of the portfolio.
- **Database craft (from the OPD playbook):** `EXPLAIN (ANALYZE, BUFFERS)` on every query you write, from Loop 2 onward; enable `pg_stat_statements` in Loop 5 and review the top 10 by total time weekly; watch `n_dead_tup` on the outbox table once it exists — a high-churn insert/update/delete table is where autovacuum bites first in every real system.
- **Build → break → fix → ADR (from the OPD playbook):** inside each loop, write the failing test or load test *before* the fix, and end the loop with a one-page ADR (context, options, decision, consequences) in `docs/adr/`. From Loop 4 on, maintain `make demo`: one command that runs a scripted chaos scenario and prints a scorecard that **reconciles** — `pings sent = 120,000 · dropped (QoS 0 / invalid) = 1,214 · duplicates received = 3,880 · duplicates persisted = 0 · persisted unique = 118,786 ✓ · breach episodes = 3 · alerts = 3 · verdict mismatches = 0`. Note it does *not* claim sent = persisted: QoS 0 pings are allowed to drop and the simulator injects duplicates on purpose, so the honest claim is "every accepted ping persisted exactly once, every episode alerted exactly once." A scorecard that claims perfection is a scorecard nobody trusts. That reconciled line is your killer demo.
- **README-driven:** keep the README current every loop; a reviewer should understand the system in 3 minutes.
- **`NOTES.md` per loop:** what broke, what surprised you, what you'd change. This is your interview-story bank and blog material. Frame at least one entry as a real incident postmortem — detection, diagnosis, fix, prevention — because "owns reliability, has the scars" is exactly what on-call-culture teams screen for.
- **Visible AI workflow:** "uses AI tools well" is a first-class criterion in every Halter JD (they name Claude, Claude Code, and Cursor). Keep a `CLAUDE.md` in the repo, use AI tools deliberately, and note in the README where they accelerated you *and where you didn't trust them and verified by hand*. Having an AI-assisted process is common; being able to show it, with judgment, is rare.
- **Interview mapping:** Loop 1 = concurrency questions; Loop 4 = delivery-semantics questions; Loop 5 = data-modeling questions; Loop 7 = "tell me about a performance problem you debugged"; Loop 8–9 = every infra question. When you finish a loop, rehearse its Explain test out loud once.
- **Don't parallelize loops.** The spiral works because each loop's pain motivates the next loop's tool. Trust it.
