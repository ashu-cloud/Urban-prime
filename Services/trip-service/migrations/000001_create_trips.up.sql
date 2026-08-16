CREATE TABLE IF NOT EXISTS trips (
    trip_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id         UUID NOT NULL,
    driver_id        UUID,
    status           TEXT NOT NULL DEFAULT 'REQUESTED',
    pickup_lat       DOUBLE PRECISION NOT NULL,
    pickup_lng       DOUBLE PRECISION NOT NULL,
    pickup_address   TEXT,
    dropoff_lat      DOUBLE PRECISION NOT NULL,
    dropoff_lng      DOUBLE PRECISION NOT NULL,
    dropoff_address  TEXT,
    vehicle_type     TEXT NOT NULL DEFAULT 'SEDAN',
    distance_km      DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    duration_secs    BIGINT NOT NULL DEFAULT 0,
    base_fare_cents  BIGINT NOT NULL DEFAULT 0,
    surge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    final_fare_cents BIGINT NOT NULL DEFAULT 0,
    currency         TEXT NOT NULL DEFAULT 'INR',
    payment_method_id TEXT,
    stripe_hold_id   TEXT,
    saga_log         JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trips_rider_id ON trips(rider_id);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id ON trips(driver_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
