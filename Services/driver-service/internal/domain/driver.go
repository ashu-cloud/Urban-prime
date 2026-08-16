package domain

import (
	"fmt"
	"time"
)

// DriverStatus represents the current state of a driver in the platform
type DriverStatus string

const (
	StatusOffline   DriverStatus = "OFFLINE"
	StatusAvailable DriverStatus = "AVAILABLE"
	StatusOnTrip    DriverStatus = "ON_TRIP"
)

// Driver represents the core domain model for a registered cab driver
type Driver struct {
	ID           string       `json:"driver_id"`
	Name         string       `json:"name"`
	Phone        string       `json:"phone"`
	Email        string       `json:"email,omitempty"`
	Status       DriverStatus `json:"status"`
	VehicleType  string       `json:"vehicle_type"`  // "SEDAN", "SUV", "PREMIUM", "BIKE"
	VehiclePlate string       `json:"vehicle_plate"`
	VehicleModel string       `json:"vehicle_model,omitempty"`
	Rating       float64      `json:"rating"`
	TotalTrips   int32        `json:"total_trips"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// CanTransitionTo enforces valid driver state transitions
func (d *Driver) CanTransitionTo(next DriverStatus) bool {
	switch d.Status {
	case StatusOffline:
		return next == StatusAvailable
	case StatusAvailable:
		return next == StatusOnTrip || next == StatusOffline
	case StatusOnTrip:
		return next == StatusAvailable || next == StatusOffline
	default:
		return false
	}
}

// ValidateTransition validates state transition and returns error if forbidden
func (d *Driver) ValidateTransition(next DriverStatus) error {
	if !d.CanTransitionTo(next) {
		return fmt.Errorf("invalid driver status transition from %s to %s for driver %s", d.Status, next, d.ID)
	}
	return nil
}
