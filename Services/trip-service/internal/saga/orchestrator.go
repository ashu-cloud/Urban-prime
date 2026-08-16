package saga

import (
	"context"
	"fmt"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/cab-booking/trip-service/internal/domain"
	"github.com/cab-booking/trip-service/internal/kafka"
	"github.com/cab-booking/trip-service/internal/osrm"
	"github.com/cab-booking/trip-service/internal/pricing"
	"github.com/cab-booking/trip-service/internal/repository"
	"github.com/google/uuid"
)

// Orchestrator is the core Saga Orchestrator engine.
// It manages distributed transaction steps for a trip across multiple microservices.
// In Go, structs store state/references, and methods attached to structs define behavior.
type Orchestrator struct {
	repo       *repository.TripRepository // Data persistence layer
	osrmClient *osrm.Client               // Road routing client
	calculator *pricing.Calculator         // Fare calculation engine
	producer   *kafka.Producer            // Async event publisher
}

// NewOrchestrator is a Constructor Function.
// Go doesn't have standard OOP class constructors, so standard idiom is to write functions named New<StructName>.
func NewOrchestrator(
	repo *repository.TripRepository,
	osrmClient *osrm.Client,
	calculator *pricing.Calculator,
	producer *kafka.Producer,
) *Orchestrator {
	return &Orchestrator{
		repo:       repo,
		osrmClient: osrmClient,
		calculator: calculator,
		producer:   producer,
	}
}

// CreateTripCmd is a Data Transfer Object (DTO) holding raw user parameters from gRPC request.
type CreateTripCmd struct {
	RiderID         string
	Pickup          domain.Location
	Dropoff         domain.Location
	VehicleType     string
	PaymentMethodID string
}

// ExecuteCreateTripSaga runs the complete Saga for initiating a new ride request.
//
// WHAT IS THE SAGA PATTERN?
// In microservices, we cannot use database transactions (ACID / 2PC) across multiple services/databases.
// Saga breaks a transaction into sequential steps. If any step fails, compensating actions are executed
// in reverse order to rollback previous steps and maintain consistency.
func (s *Orchestrator) ExecuteCreateTripSaga(ctx context.Context, cmd CreateTripCmd) (*domain.Trip, error) {
	// Generate a unique UUID v4 for the new trip (e.g., "550e8400-e29b-41d4-a716-446655440000")
	tripID := uuid.New().String()
	now := time.Now()

	logger.Info(ctx, "Saga Step 1: Querying OSRM for routing & computing fare", "trip_id", tripID)

	// STEP 1: ROUTING (Call OSRM HTTP Engine)
	// Calculates real driving distance (km) and estimated duration (seconds) between pickup & dropoff coordinates
	route, err := s.osrmClient.GetRoute(ctx, cmd.Pickup, cmd.Dropoff)
	if err != nil {
		logger.Error(ctx, "OSRM routing failed", "error", err)
		return nil, fmt.Errorf("failed to calculate trip route: %w", err)
	}

	// STEP 2: FARE CALCULATION (Uber Formula)
	// Base fare + (per_km_rate * distance_km) + (per_min_rate * duration_mins) * surge_multiplier
	breakdown := s.calculator.CalculateFare(route.DistanceKm, route.DurationSecs, 1.0)

	// STEP 3: CONSTRUCT TRIP DOMAIN ENTITY
	trip := &domain.Trip{
		ID:              tripID,
		RiderID:         cmd.RiderID,
		Status:          domain.StatusRequested, // Initial status: REQUESTED
		Pickup:          cmd.Pickup,
		Dropoff:         cmd.Dropoff,
		VehicleType:     cmd.VehicleType,
		DistanceKm:      route.DistanceKm,
		DurationSecs:    route.DurationSecs,
		BaseFareCents:   breakdown.BaseFareCents,
		SurgeMultiplier: breakdown.SurgeMultiplier,
		FinalFareCents:  breakdown.TotalFareCents,
		Currency:        "INR",
		PaymentMethodID: cmd.PaymentMethodID,
		SagaLog: []domain.SagaStepLog{
			{
				StepName:  "CREATE_TRIP_PENDING",
				Status:    "SUCCESS",
				Details:   fmt.Sprintf("Route calculated: %.2f km, fare: ₹%.2f", route.DistanceKm, float64(breakdown.TotalFareCents)/100.0),
				Timestamp: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// STEP 4: PERSIST TO POSTGRESQL DATABASE
	// Inserts trip row into PostgreSQL table 'trips' with JSONB audit saga_log
	if err := s.repo.Create(ctx, trip); err != nil {
		logger.Error(ctx, "Failed to persist trip to DB", "error", err)
		return nil, err
	}

	// STEP 5: PUBLISH EVENT TO KAFKA
	// Emits `TripCreated` payload to Kafka topic `trip.events.v1` for downstream services (Notifications, Analytics)
	eventPayload := kafka.TripEventPayload{
		TripID:        trip.ID,
		RiderID:       trip.RiderID,
		Status:        string(trip.Status),
		PickupLat:     trip.Pickup.Latitude,
		PickupLng:     trip.Pickup.Longitude,
		DropoffLat:    trip.Dropoff.Latitude,
		DropoffLng:    trip.Dropoff.Longitude,
		DistanceKm:    trip.DistanceKm,
		EstimatedFare: trip.FinalFareCents,
		Currency:      trip.Currency,
		Timestamp:     now,
	}

	if err := s.producer.PublishTripEvent(ctx, "trip.events.v1", eventPayload); err != nil {
		logger.Warn(ctx, "Publishing trip created event returned warning", "error", err)
	}

	// STEP 6: TRANSITION STATE TO 'MATCHING'
	// State Machine Transition: REQUESTED -> MATCHING
	// Driver Service will listen to this event and begin searching for nearby available drivers.
	matchingStep := domain.SagaStepLog{
		StepName:  "TRANSITION_MATCHING",
		Status:    "SUCCESS",
		Details:   "Trip ready for driver matchmaking search",
		Timestamp: time.Now(),
	}

	// If database state update fails, execute COMPENSATING TRANSACTION to rollback!
	if err := s.repo.UpdateStatus(ctx, trip.ID, domain.StatusMatching, matchingStep); err != nil {
		logger.Error(ctx, "Saga Compensation Triggered: Failed to update status to MATCHING", "error", err)
		s.CompensateTripCreation(ctx, trip.ID, "Failed state transition to MATCHING")
		return nil, fmt.Errorf("saga step 2 failed: %w", err)
	}

	trip.Status = domain.StatusMatching
	trip.SagaLog = append(trip.SagaLog, matchingStep)

	// Publish `TripMatchingStarted` Event to Kafka
	eventPayload.Status = string(domain.StatusMatching)
	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", eventPayload)

	logger.Info(ctx, "Saga Completed Successfully: Trip created & in MATCHING state", "trip_id", trip.ID, "fare_inr", float64(trip.FinalFareCents)/100.0)

	return trip, nil
}

// CompensateTripCreation is the COMPENSATING TRANSACTION function.
// If any downstream step fails (e.g. payment hold fails or no drivers accept), this function is invoked
// to mark the trip as CANCELLED, append compensation audit log, and broadcast cancellation to Kafka.
func (s *Orchestrator) CompensateTripCreation(ctx context.Context, tripID string, reason string) {
	logger.Warn(ctx, "Executing Saga Compensating Transaction for trip", "trip_id", tripID, "reason", reason)

	compStep := domain.SagaStepLog{
		StepName:  "COMPENSATE_CANCEL_TRIP",
		Status:    "COMPENSATED",
		Details:   reason,
		Timestamp: time.Now(),
	}

	// Revert status to CANCELLED in DB
	_ = s.repo.UpdateStatus(ctx, tripID, domain.StatusCancelled, compStep)

	// Publish cancellation event to Kafka
	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", kafka.TripEventPayload{
		TripID:    tripID,
		Status:    string(domain.StatusCancelled),
		Timestamp: time.Now(),
	})
}
