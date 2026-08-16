CREATE TABLE IF NOT EXISTS drivers (
    driver_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    phone         TEXT NOT NULL UNIQUE,
    email         TEXT,
    status        TEXT NOT NULL DEFAULT 'OFFLINE',
    vehicle_type  TEXT NOT NULL DEFAULT 'SEDAN',
    vehicle_plate TEXT NOT NULL,
    vehicle_model TEXT,
    rating        DOUBLE PRECISION NOT NULL DEFAULT 5.0,
    total_trips   INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);
CREATE INDEX IF NOT EXISTS idx_drivers_vehicle_type ON drivers(vehicle_type);
