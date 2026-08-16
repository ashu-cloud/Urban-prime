package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cab-booking/driver-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DriverRepository encapsulates raw SQL database operations for driver profiles in PostgreSQL.
type DriverRepository struct {
	pool *pgxpool.Pool // Pointer to shared PostgreSQL connection pool
}

// NewDriverRepository constructs DriverRepository instance
func NewDriverRepository(pool *pgxpool.Pool) *DriverRepository {
	return &DriverRepository{pool: pool}
}

// CreateDriver inserts a new driver record into PostgreSQL
func (r *DriverRepository) CreateDriver(ctx context.Context, driver *domain.Driver) error {
	query := `
		INSERT INTO drivers (
			driver_id, name, phone, email, status,
			vehicle_type, vehicle_plate, vehicle_model,
			rating, total_trips, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12
		)
	`

	_, err := r.pool.Exec(ctx, query,
		driver.ID, driver.Name, driver.Phone, driver.Email, string(driver.Status),
		driver.VehicleType, driver.VehiclePlate, driver.VehicleModel,
		driver.Rating, driver.TotalTrips, driver.CreatedAt, driver.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("db create driver failed: %w", err)
	}

	return nil
}

// GetByID fetches a driver profile by UUID
func (r *DriverRepository) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	query := `
		SELECT 
			driver_id, name, phone, COALESCE(email, ''), status,
			vehicle_type, vehicle_plate, COALESCE(vehicle_model, ''),
			rating, total_trips, created_at, updated_at
		FROM drivers
		WHERE driver_id = $1
	`

	var d domain.Driver
	var statusStr string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.Phone, &d.Email, &statusStr,
		&d.VehicleType, &d.VehiclePlate, &d.VehicleModel,
		&d.Rating, &d.TotalTrips, &d.CreatedAt, &d.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("driver not found: %s", id)
		}
		return nil, fmt.Errorf("db get driver failed: %w", err)
	}

	d.Status = domain.DriverStatus(statusStr)
	return &d, nil
}

// UpdateStatus updates the driver's current availability status in PostgreSQL
func (r *DriverRepository) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
	query := `
		UPDATE drivers
		SET status = $1, updated_at = $2
		WHERE driver_id = $3
	`

	res, err := r.pool.Exec(ctx, query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("db update driver status failed: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("driver not found for status update: %s", id)
	}

	return nil
}
