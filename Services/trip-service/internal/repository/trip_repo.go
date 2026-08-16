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
type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

func (r *TripRepository) Create(ctx context.Context, trip *domain.Trip) error {
	sagaJSON, err := json.Marshal(trip.SagaLog)
	if err != nil {
		return fmt.Errorf("failed to marshal saga_log: %w", err)
	}

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

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.RiderID, &t.DriverID, &statusStr,
		&t.Pickup.Latitude, &t.Pickup.Longitude, &t.Pickup.Address,
		&t.Dropoff.Latitude, &t.Dropoff.Longitude, &t.Dropoff.Address,
		&t.VehicleType, &t.DistanceKm, &t.DurationSecs,
		&t.BaseFareCents, &t.SurgeMultiplier, &t.FinalFareCents,
		&t.Currency, &t.PaymentMethodID, &t.StripeHoldID, &sagaBytes, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("trip not found: %s", id)
		}
		return nil, fmt.Errorf("db get trip failed: %w", err)
	}

	t.Status = domain.TripStatus(statusStr)

	if len(sagaBytes) > 0 {
		_ = json.Unmarshal(sagaBytes, &t.SagaLog)
	}

	return &t, nil
}

func (r *TripRepository) UpdateStatus(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	stepJSON, err := json.Marshal([]domain.SagaStepLog{step})
	if err != nil {
		return fmt.Errorf("failed to marshal saga step: %w", err)
	}

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

	if res.RowsAffected() == 0 {
		return fmt.Errorf("trip not found for status update: %s", id)
	}

	return nil
}

// AssignDriver atomically sets driver_id, updates status to ASSIGNED, and appends saga step log
func (r *TripRepository) AssignDriver(ctx context.Context, tripID, driverID string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	stepJSON, err := json.Marshal([]domain.SagaStepLog{step})
	if err != nil {
		return fmt.Errorf("failed to marshal saga step: %w", err)
	}

	query := `
		UPDATE trips
		SET 
			driver_id = $1,
			status = $2,
			saga_log = saga_log || $3::jsonb,
			updated_at = $4
		WHERE trip_id = $5
	`

	res, err := r.pool.Exec(ctx, query, driverID, string(newStatus), stepJSON, time.Now(), tripID)
	if err != nil {
		return fmt.Errorf("db assign driver failed: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("trip not found for driver assignment: %s", tripID)
	}

	return nil
}
