# 2. --speed scales physics, never the clock

Status: accepted

## Context

Watching one simulated day takes one real day. That makes every interesting
behaviour — a herd drifting to the fence, a full audio → vibration → pulse
ladder, cows wandering back — untestable in a work session. We need
`--speed=60`: 60× the movement per real second.

But a simulation has three different "times", and they must not be mixed:

| time | what it is | `--speed` touches it? |
|---|---|---|
| **wall clock** | the timestamp stamped on each ping | **never** |
| **physics dt** | how far a cow moves per step | yes |
| **ticker interval** | how often the loop ticks (1 s) | no |

## Options

**(a) Multiply `dt` by speed.** One step of 60 s instead of 60 steps of 1 s.
Cheap, but wrong: heading diffusion grows like √t, not t. A 60× turn per
step spins cows ~3 full circles per tick; leaving the turn alone understates
it 60×. Both break the walk from Loop 0 in opposite directions.

**(b) Multiply timestamps by speed.** Pings carry simulated time, so a fast
run looks "from the future". This breaks Loop 4 before it exists: ingest
rejects future-dated pings as spoofed or bad-clocked devices, and our own
simulator would be its biggest offender. Stale-collar minutes, alert
cooldowns, retention windows — every wall-clock threshold dies with it.

**(c) Many small steps, real timestamps.** Each tick runs `speed` × 1-second
physics substeps and stamps pings with the real wall-clock time.

## Decision

(c).

- `Run` does `for i := 0; i < Speed; i++ { cow.Step(time.Second, center) }`.
  Physics at speed 60 is bit-for-bit what 60 real-time seconds would do.
  Cost is 60× compute per tick — nothing at 50 cows.
- Timestamps come from a `Clock` interface (`Now() time.Time`), real in
  production, stubbed in tests. **Production code never calls `time.Now()`
  directly**, so speed can never fast-forward a timestamp by accident.
- Anything counted in ticks (collar dwell, escalation every 8 ticks) stays
  in ticks. Anything counted in minutes (future: stale collars, cooldowns)
  stays in wall-clock. The two never meet.

## Consequences

- `--speed=60 --seed=42` twice yields the same ping sequence
  (`TestSeedReproducesBreachSequence`). Determinism is over seed + tick
  count, not wall time: a tick landing exactly on shutdown is a coin-flip,
  by design.
- A simulated grazing day runs in 24 real minutes at speed 60.
- Collar dwell is speed-independent *in ticks* but speed-compressed *in
  wall time*: the 8-tick ladder from audio to vibration takes 8 s at any
  speed. Cooldowns, when they arrive, must therefore be wall-clock.
- Loop 4's future-ping rejection is safe against our own simulator.
