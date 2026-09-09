-- +goose Up
CREATE TABLE latest (
  farm_id       TEXT NOT NULL,
  cow_id        TEXT NOT NULL,
  x             DOUBLE PRECISION NOT NULL,
  y             DOUBLE PRECISION NOT NULL,
  state         TEXT NOT NULL,
  cue           TEXT NOT NULL,
  activity      TEXT NOT NULL,
  at            TIMESTAMPTZ NOT NULL,
  fence_version INT NOT NULL DEFAULT 1,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (farm_id, cow_id)
);

CREATE TABLE alerts (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  farm_id    TEXT NOT NULL,
  cow_id     TEXT NOT NULL,
  state      TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  ended_at   TIMESTAMPTZ,
  CONSTRAINT open_ended_check CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE UNIQUE INDEX alerts_one_open_per_cow ON alerts (farm_id, cow_id) WHERE ended_at IS NULL;
CREATE INDEX alerts_farm_started ON alerts (farm_id, started_at DESC);

CREATE TABLE fences (
  farm_id    TEXT NOT NULL,
  paddock_id TEXT NOT NULL DEFAULT 'p1',
  version    INT NOT NULL,
  polygon    JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (farm_id, paddock_id, version)
);

-- +goose Down
DROP TABLE IF EXISTS latest;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS fences;
