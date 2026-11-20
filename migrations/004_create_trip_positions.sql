-- +goose NO TRANSACTION
-- +goose Up
ALTER TABLE trips ADD COLUMN IF NOT EXISTS last_position_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS trip_positions (
    id          BIGSERIAL PRIMARY KEY,
    trip_id     UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    latitude    DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude   DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS trip_positions_trip_recorded_idx
    ON trip_positions (trip_id, recorded_at, id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS trips_stale_positions_idx
    ON trips ((COALESCE(last_position_at, started_at)), id)
    WHERE status = 'active';

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS trips_stale_positions_idx;
DROP INDEX CONCURRENTLY IF EXISTS trip_positions_trip_recorded_idx;
DROP TABLE IF EXISTS trip_positions;
ALTER TABLE trips DROP COLUMN IF EXISTS last_position_at;
