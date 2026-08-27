package handler

import (
	"context"

	"github.com/cab-booking/pkg/logger"
	tripv1 "github.com/cab-booking/proto/gen/trip/v1"
	"github.com/cab-booking/trip-service/internal/domain"
	"github.com/cab-booking/trip-service/internal/saga"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TripLookup interface {
	GetByID(ctx context.Context, id string) (*domain.Trip, error)
}

// TripHandler implements the protobuf-generated `tripv1.TripServiceServer` interface.
// It acts as the Controller layer in gRPC services, receiving binary gRPC requests,
// validating arguments, triggering business logic (Saga), and returning gRPC responses.
type TripHandler struct {
	tripv1.UnimplementedTripServiceServer // Embedded struct providing default unimplemented methods
	orchestrator *saga.Orchestrator        // Saga Orchestrator engine
	repo         TripLookup                // Database repository
}

// NewTripHandler constructs a new gRPC TripHandler instance
func NewTripHandler(orchestrator *saga.Orchestrator, repo TripLookup) *TripHandler {
	return &TripHandler{
		orchestrator: orchestrator,
		repo:         repo,
	}
}

// CreateTrip is the gRPC RPC handler function for initiating a new trip request.
func (h *TripHandler) CreateTrip(ctx context.Context, req *tripv1.CreateTripRequest) (*tripv1.CreateTripResponse, error) {
	// 1. INPUT VALIDATION: Ensure required fields are provided
	if req.RiderId == "" {
		// status.Error returns standard gRPC error with status code (codes.InvalidArgument = HTTP 400)
		return nil, status.Error(codes.InvalidArgument, "rider_id is required")
	}
	if req.PickupLocation == nil || req.DropoffLocation == nil {
		return nil, status.Error(codes.InvalidArgument, "pickup_location and dropoff_location are required")
	}
	if !validCoordinates(req.PickupLocation.Latitude, req.PickupLocation.Longitude) {
		return nil, status.Error(codes.InvalidArgument, "pickup coordinates are invalid")
	}
	if !validCoordinates(req.DropoffLocation.Latitude, req.DropoffLocation.Longitude) {
		return nil, status.Error(codes.InvalidArgument, "dropoff coordinates are invalid")
	}

	// 2. CONVERT PROTOBUF REQUEST DTO -> DOMAIN COMMAND STRUCT
	cmd := saga.CreateTripCmd{
		RiderID: req.RiderId,
		Pickup: domain.Location{
			Latitude:  req.PickupLocation.Latitude,
			Longitude: req.PickupLocation.Longitude,
			Address:   req.PickupLocation.Address,
		},
		Dropoff: domain.Location{
			Latitude:  req.DropoffLocation.Latitude,
			Longitude: req.DropoffLocation.Longitude,
			Address:   req.DropoffLocation.Address,
		},
		VehicleType:     req.VehicleType,
		PaymentMethodID: req.PaymentMethodId,
	}

	// 3. EXECUTE SAGA ORCHESTRATOR
	trip, err := h.orchestrator.ExecuteCreateTripSaga(ctx, cmd)
	if err != nil {
		logger.Error(ctx, "CreateTrip gRPC call failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create trip: %v", err)
	}

	// 4. MAP DOMAIN STRUCT -> PROTOBUF RESPONSE & RETURN
	return &tripv1.CreateTripResponse{
		Trip: mapDomainTripToProto(trip),
	}, nil
}

// GetTrip handles RPC requests to fetch existing trip details by UUID
func (h *TripHandler) GetTrip(ctx context.Context, req *tripv1.GetTripRequest) (*tripv1.GetTripResponse, error) {
	if req.TripId == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}

	trip, err := h.repo.GetByID(ctx, req.TripId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "trip not found: %v", err)
	}

	return &tripv1.GetTripResponse{
		Trip: mapDomainTripToProto(trip),
	}, nil
}

// CancelTrip handles RPC requests to cancel an active trip request
func (h *TripHandler) CancelTrip(ctx context.Context, req *tripv1.CancelTripRequest) (*tripv1.CancelTripResponse, error) {
	if req.TripId == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}

	// Fetch trip record
	trip, err := h.repo.GetByID(ctx, req.TripId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "trip not found: %v", err)
	}

	// Validate state machine rule: can this trip be cancelled in its current state?
	if err := trip.ValidateTransition(domain.StatusCancelled); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}

	reason := req.Reason
	if reason == "" {
		reason = "Cancelled by user"
	}

	// Trigger Saga Compensating Action to rollback resources and mark trip CANCELLED
	h.orchestrator.CompensateTripCreation(ctx, req.TripId, reason)

	// Fetch updated state
	updatedTrip, err := h.repo.GetByID(ctx, req.TripId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated trip: %v", err)
	}

	return &tripv1.CancelTripResponse{
		Success: true,
		Trip:    mapDomainTripToProto(updatedTrip),
	}, nil
}

// Helper function to map internal domain.Trip model -> protobuf tripv1.Trip model
func mapDomainTripToProto(t *domain.Trip) *tripv1.Trip {
	driverID := ""
	if t.DriverID != nil {
		driverID = *t.DriverID
	}

	// Map domain status to Protobuf Enum value
	statusEnum := tripv1.TripStatus_TRIP_STATUS_UNSPECIFIED
	switch t.Status {
	case domain.StatusRequested:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_REQUESTED
	case domain.StatusMatching:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_MATCHING
	case domain.StatusAssigned:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_ASSIGNED
	case domain.StatusInProgress:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_IN_PROGRESS
	case domain.StatusCompleted:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_COMPLETED
	case domain.StatusCancelled, domain.StatusCancelledNoDriver:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_CANCELLED
	case domain.StatusPaymentFailed:
		statusEnum = tripv1.TripStatus_TRIP_STATUS_PAYMENT_FAILED
	}

	return &tripv1.Trip{
		TripId:   t.ID,
		RiderId:  t.RiderID,
		DriverId: driverID,
		PickupLocation: &tripv1.Location{
			Latitude:  t.Pickup.Latitude,
			Longitude: t.Pickup.Longitude,
			Address:   t.Pickup.Address,
		},
		DropoffLocation: &tripv1.Location{
			Latitude:  t.Dropoff.Latitude,
			Longitude: t.Dropoff.Longitude,
			Address:   t.Dropoff.Address,
		},
		Status: statusEnum,
		EstimatedFare: &tripv1.Money{
			Currency:    t.Currency,
			AmountCents: t.FinalFareCents,
		},
		DistanceKm:               t.DistanceKm,
		EstimatedDurationSeconds: t.DurationSecs,
		CreatedAt:                t.CreatedAt.Unix(),
		UpdatedAt:                t.UpdatedAt.Unix(),
	}
}

func validCoordinates(lat, lng float64) bool {
	if lat == 0 && lng == 0 {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
