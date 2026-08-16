package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cab-booking/trip-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TripRepository encapsulates raw SQL database queries for the 'trips' table in PostgreSQL.
// Senior engineers prefer raw SQL with `pgx` over heavy ORMs like GORM for maximum performance,
// precise query control, and zero hidden magic.
type TripRepository struct {
	pool *pgxpool.Pool // Pointer to shared PostgreSQL connection pool
}

// NewTripRepository is constructor function for TripRepository
func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

// Create executes an INSERT query to store a new trip record in PostgreSQL
func (r *TripRepository) Create(ctx context.Context, trip *domain.Trip) error {
	// Marshal the Go slice `[]SagaStepLog` into a raw JSON byte array for the PostgreSQL JSONB column `saga_log`
	sagaJSON, err := json.Marshal(trip.SagaLog)
	if err != nil {
		return fmt.Errorf("failed to marshal saga_log: %w", err)
	}

	// Parameterized SQL Query: $1, $2, $3... placeholders prevent SQL Injection vulnerabilities!
	query := `
		INSERT INTO trips (
			trip_id, rider_id, driver_id, status,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			vehicle_type, distance_km, duration_secs,
			base_fare_cents, surge_multiplier, final_fare_cents,
			currency, payment_method_id, stripe_hold_id, saga_log, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18, $19, $20, $21, $22
		)
	`

	// r.pool.Exec executes the query against PostgreSQL pool
	_, err = r.pool.Exec(ctx, query,
		trip.ID, trip.RiderID, trip.DriverID, string(trip.Status),
		trip.Pickup.Latitude, trip.Pickup.Longitude, trip.Pickup.Address,
		trip.Dropoff.Latitude, trip.Dropoff.Longitude, trip.Dropoff.Address,
		trip.VehicleType, trip.DistanceKm, trip.DurationSecs,
		trip.BaseFareCents, trip.SurgeMultiplier, trip.FinalFareCents,
		trip.Currency, trip.PaymentMethodID, trip.StripeHoldID, sagaJSON, trip.CreatedAt, trip.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("db create trip failed: %w", err)
	}

	return nil
}

// GetByID queries a single trip from PostgreSQL by UUID
func (r *TripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	query := `
		SELECT 
			trip_id, rider_id, driver_id, status,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			vehicle_type, distance_km, duration_secs,
			base_fare_cents, surge_multiplier, final_fare_cents,
			currency, payment_method_id, stripe_hold_id, saga_log, created_at, updated_at
		FROM trips
		WHERE trip_id = $1
	`

	var t domain.Trip
	var statusStr string
	var sagaBytes []byte

	// QueryRow returns a single row. Scan() maps database column values directly to Go variables.
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.RiderID, &t.DriverID, &statusStr,
		&t.Pickup.Latitude, &t.Pickup.Longitude, &t.Pickup.Address,
		&t.Dropoff.Latitude, &t.Dropoff.Longitude, &t.Dropoff.Address,
		&t.VehicleType, &t.DistanceKm, &t.DurationSecs,
		&t.BaseFareCents, &t.SurgeMultiplier, &t.FinalFareCents,
		&t.Currency, &t.PaymentMethodID, &t.StripeHoldID, &sagaBytes, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		// Check if error is 'pgx.ErrNoRows' (meaning 0 matching rows found for UUID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("trip not found: %s", id)
		}
		return nil, fmt.Errorf("db get trip failed: %w", err)
	}

	// Convert string status back to domain.TripStatus type
	t.Status = domain.TripStatus(statusStr)

	// Unmarshal raw JSONB bytes into Go slice []SagaStepLog
	if len(sagaBytes) > 0 {
		_ = json.Unmarshal(sagaBytes, &t.SagaLog)
	}

	return &t, nil
}

// UpdateStatus atomically updates the trip status and appends a step entry to the JSONB saga_log column
func (r *TripRepository) UpdateStatus(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	stepJSON, err := json.Marshal([]domain.SagaStepLog{step})
	if err != nil {
		return fmt.Errorf("failed to marshal saga step: %w", err)
	}

	// PostgreSQL JSONB Concatenation Operator (`||`):
	// Appends new step JSON object to existing JSONB array in database without rewriting full log!
	query := `
		UPDATE trips
		SET 
			status = $1,
			saga_log = saga_log || $2::jsonb,
			updated_at = $3
		WHERE trip_id = $4
	`

	res, err := r.pool.Exec(ctx, query, string(newStatus), stepJSON, time.Now(), id)
	if err != nil {
		return fmt.Errorf("db update status failed: %w", err)
	}

	// Check if any row was actually updated
	if res.RowsAffected() == 0 {
		return fmt.Errorf("trip not found for status update: %s", id)
	}

	return nil
}
