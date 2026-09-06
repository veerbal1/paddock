# 1. Work in a local plane, not GPS coordinates

Status: accepted

## Context

The system reasons in metres. Every rule is written in metres: the warning
zone starts 10 m inside the boundary, a breach requires 5 m outside, returning
to `inside` requires 15 m in, a cow walks at ~1 m/s.

GPS gives degrees. A degree is an angle, not a length, and the two axes are
not the same size — one degree of longitude is 111 km at the equator but
95 km at 31°N, shrinking by cos(latitude).

So something has to translate.

## Options

**(a) Store and compute in lat/lng.** Every distance check does spherical
trigonometry (haversine), every movement converts metres to degrees with a
cos(latitude) correction.

**(b) Local plane.** Pick a reference origin per farm. Work in flat x/y metres
everywhere inside the system. Convert to and from degrees only at the boundary.

## Decision

(b) — a local plane.

## Consequences

- **The flat model is correct here, not a shortcut.** Earth's curvature causes
  ~8 cm of deviation over 1 km. GPS error is 3–5 m. Curvature is roughly 50×
  below the noise floor, so modelling the sphere buys nothing at paddock scale.

- **No trigonometry inside the system.** Distance is subtraction for an
  axis-aligned rectangle, Pythagoras for arbitrary points. No haversine.

- **One conversion, in one place**, roughly two lines each way:
  ```
  x = (lng − originLng) × 111195 × cos(originLat)
  y = (lat − originLat) × 111195
  ```

- **A hard rule:** `geom`, `fence`, `collar` must never see a lat/lng. If they
  do, this decision is dead and converting later becomes a rewrite.

- **Each farm needs a stored origin** — a reference lat/lng that `(0,0)` maps
  to. Not needed yet; needed when GPS arrives.

- **Leaflet needs degrees** (Loop 2), so the outbound conversion must exist by
  then. Postgres/PostGIS will also store lat/lng, since 4326 is the portable
  format.

- **This breaks down at scale.** A polygon spanning hundreds of kilometres, or
  one crossing the antimeridian or a pole, needs real geodesy. Out of scope:
  paddocks are hundreds of metres.
