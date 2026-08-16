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

type Orchestrator struct {
	repo       *repository.TripRepository
	osrmClient *osrm.Client
	calculator *pricing.Calculator
	producer   *kafka.Producer
}

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

type CreateTripCmd struct {
	RiderID         string
	Pickup          domain.Location
	Dropoff         domain.Location
	VehicleType     string
	PaymentMethodID string
}

func (s *Orchestrator) ExecuteCreateTripSaga(ctx context.Context, cmd CreateTripCmd) (*domain.Trip, error) {
	tripID := uuid.New().String()
	now := time.Now()

	logger.Info(ctx, "Saga Step 1: Querying OSRM for routing & computing fare", "trip_id", tripID)

	route, err := s.osrmClient.GetRoute(ctx, cmd.Pickup, cmd.Dropoff)
	if err != nil {
		logger.Error(ctx, "OSRM routing failed", "error", err)
		return nil, fmt.Errorf("failed to calculate trip route: %w", err)
	}

	breakdown := s.calculator.CalculateFare(route.DistanceKm, route.DurationSecs, 1.0)

	trip := &domain.Trip{
		ID:              tripID,
		RiderID:         cmd.RiderID,
		Status:          domain.StatusRequested,
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

	if err := s.repo.Create(ctx, trip); err != nil {
		logger.Error(ctx, "Failed to persist trip to DB", "error", err)
		return nil, err
	}

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

	matchingStep := domain.SagaStepLog{
		StepName:  "TRANSITION_MATCHING",
		Status:    "SUCCESS",
		Details:   "Trip ready for driver matchmaking search",
		Timestamp: time.Now(),
	}

	if err := s.repo.UpdateStatus(ctx, trip.ID, domain.StatusMatching, matchingStep); err != nil {
		logger.Error(ctx, "Saga Compensation Triggered: Failed to update status to MATCHING", "error", err)
		s.CompensateTripCreation(ctx, trip.ID, "Failed state transition to MATCHING")
		return nil, fmt.Errorf("saga step 2 failed: %w", err)
	}

	trip.Status = domain.StatusMatching
	trip.SagaLog = append(trip.SagaLog, matchingStep)

	eventPayload.Status = string(domain.StatusMatching)
	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", eventPayload)

	logger.Info(ctx, "Saga Completed Successfully: Trip created & in MATCHING state", "trip_id", trip.ID, "fare_inr", float64(trip.FinalFareCents)/100.0)

	return trip, nil
}

// AssignDriverToTrip updates trip state machine to ASSIGNED and records matched driver_id
func (s *Orchestrator) AssignDriverToTrip(ctx context.Context, tripID, driverID string) error {
	step := domain.SagaStepLog{
		StepName:  "DRIVER_MATCHED_ASSIGNED",
		Status:    "SUCCESS",
		Details:   fmt.Sprintf("Matched and assigned driver %s", driverID),
		Timestamp: time.Now(),
	}

	if err := s.repo.AssignDriver(ctx, tripID, driverID, domain.StatusAssigned, step); err != nil {
		return fmt.Errorf("failed to assign driver in repository: %w", err)
	}

	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", kafka.TripEventPayload{
		TripID:    tripID,
		Status:    string(domain.StatusAssigned),
		Timestamp: time.Now(),
	})

	return nil
}

// CompensateNoDriverAvailable is executed when driver dispatch loop finishes with no driver acceptance
func (s *Orchestrator) CompensateNoDriverAvailable(ctx context.Context, tripID string) {
	logger.Warn(ctx, "Executing Saga Compensation: Reverting trip status to CANCELLED_NO_DRIVER", "trip_id", tripID)

	step := domain.SagaStepLog{
		StepName:  "COMPENSATE_NO_DRIVER_AVAILABLE",
		Status:    "COMPENSATED",
		Details:   "All nearby candidate drivers declined or timed out",
		Timestamp: time.Now(),
	}

	_ = s.repo.UpdateStatus(ctx, tripID, domain.StatusCancelledNoDriver, step)

	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", kafka.TripEventPayload{
		TripID:    tripID,
		Status:    string(domain.StatusCancelledNoDriver),
		Timestamp: time.Now(),
	})
}

func (s *Orchestrator) CompensateTripCreation(ctx context.Context, tripID string, reason string) {
	logger.Warn(ctx, "Executing Saga Compensating Transaction for trip", "trip_id", tripID, "reason", reason)

	compStep := domain.SagaStepLog{
		StepName:  "COMPENSATE_CANCEL_TRIP",
		Status:    "COMPENSATED",
		Details:   reason,
		Timestamp: time.Now(),
	}

	_ = s.repo.UpdateStatus(ctx, tripID, domain.StatusCancelled, compStep)

	_ = s.producer.PublishTripEvent(ctx, "trip.events.v1", kafka.TripEventPayload{
		TripID:    tripID,
		Status:    string(domain.StatusCancelled),
		Timestamp: time.Now(),
	})
}
