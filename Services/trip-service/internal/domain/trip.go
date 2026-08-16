package domain

import (
	"fmt"
	"time"
)

// TripStatus defines a custom type for trip state machine states.
// In Go, custom type aliases on strings are used to implement Enums safely.
type TripStatus string

// Enum values representing all valid trip state machine statuses
const (
	StatusRequested         TripStatus = "REQUESTED"
	StatusMatching          TripStatus = "MATCHING"
	StatusAssigned          TripStatus = "ASSIGNED"
	StatusInProgress        TripStatus = "IN_PROGRESS"
	StatusCompleted         TripStatus = "COMPLETED"
	StatusCancelled         TripStatus = "CANCELLED"
	StatusCancelledNoDriver TripStatus = "CANCELLED_NO_DRIVER"
	StatusPaymentFailed     TripStatus = "PAYMENT_FAILED"
)

// Location represents a geographic coordinate and human-readable address
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// SagaStepLog holds audit history for a single Saga transaction step.
// Stored as a JSON array inside the PostgreSQL JSONB column `saga_log`.
type SagaStepLog struct {
	StepName  string    `json:"step_name"`
	Status    string    `json:"status"` // "SUCCESS", "FAILED", "COMPENSATED"
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Trip is the Core Domain Entity representing a cab booking trip
type Trip struct {
	ID              string        `json:"trip_id"`
	RiderID         string        `json:"rider_id"`
	DriverID        *string       `json:"driver_id,omitempty"` // Pointer (*string) allows nil when no driver assigned yet!
	Status          TripStatus    `json:"status"`
	Pickup          Location      `json:"pickup_location"`
	Dropoff         Location      `json:"dropoff_location"`
	VehicleType     string        `json:"vehicle_type"`
	DistanceKm      float64       `json:"distance_km"`
	DurationSecs    int64         `json:"duration_secs"`
	BaseFareCents   int64         `json:"base_fare_cents"`
	SurgeMultiplier float64       `json:"surge_multiplier"`
	FinalFareCents  int64         `json:"final_fare_cents"`
	Currency        string        `json:"currency"`
	PaymentMethodID string        `json:"payment_method_id,omitempty"`
	StripeHoldID    *string       `json:"stripe_hold_id,omitempty"`
	SagaLog         []SagaStepLog `json:"saga_log"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// CanTransitionTo enforces strict Trip State Machine transition rules.
// Method syntax in Go: `(t *Trip)` is a pointer receiver attaching this method to struct `Trip`.
func (t *Trip) CanTransitionTo(next TripStatus) bool {
	switch t.Status {
	case StatusRequested:
		return next == StatusMatching || next == StatusPaymentFailed || next == StatusCancelled
	case StatusMatching:
		return next == StatusAssigned || next == StatusCancelledNoDriver || next == StatusCancelled
	case StatusAssigned:
		return next == StatusInProgress || next == StatusCancelled
	case StatusInProgress:
		return next == StatusCompleted || next == StatusCancelled
	default:
		return false
	}
}

// ValidateTransition checks if state transition is allowed and returns detailed error if invalid
func (t *Trip) ValidateTransition(next TripStatus) error {
	if !t.CanTransitionTo(next) {
		return fmt.Errorf("invalid state transition from %s to %s for trip %s", t.Status, next, t.ID)
	}
	return nil
}
