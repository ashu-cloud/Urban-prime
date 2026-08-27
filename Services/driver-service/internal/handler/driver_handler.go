package handler

import (
	"context"
	"time"

	driverv1 "github.com/cab-booking/proto/gen/driver/v1"
	"github.com/cab-booking/driver-service/internal/dispatch"
	"github.com/cab-booking/driver-service/internal/domain"
	"github.com/cab-booking/driver-service/internal/geo"
	"github.com/cab-booking/pkg/logger"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DriverStore interface {
	GetByID(ctx context.Context, id string) (*domain.Driver, error)
	UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error
	CreateDriver(ctx context.Context, driver *domain.Driver) error
}

// DriverHandler implements the protobuf-generated `driverv1.DriverServiceServer` gRPC interface.
type DriverHandler struct {
	driverv1.UnimplementedDriverServiceServer
	dispatchLoop *dispatch.DispatchLoop
	geoService   *geo.GeoService
	repo         DriverStore
}

// NewDriverHandler constructs a new DriverHandler instance
func NewDriverHandler(
	dispatchLoop *dispatch.DispatchLoop,
	geoService *geo.GeoService,
	repo DriverStore,
) *DriverHandler {
	return &DriverHandler{
		dispatchLoop: dispatchLoop,
		geoService:   geoService,
		repo:         repo,
	}
}

// MatchDriver is called by Trip Service Saga Orchestrator to find and dispatch nearest driver
func (h *DriverHandler) MatchDriver(ctx context.Context, req *driverv1.MatchDriverRequest) (*driverv1.MatchDriverResponse, error) {
	if req.TripId == "" {
		return nil, status.Error(codes.InvalidArgument, "trip_id is required")
	}

	// Trigger Dispatch Loop Engine
	driver, err := h.dispatchLoop.FindAndDispatchDriver(
		ctx,
		req.TripId,
		req.PickupLatitude,
		req.PickupLongitude,
		req.VehicleType,
	)

	if err != nil {
		logger.Error(ctx, "MatchDriver dispatch error", "error", err)
		return nil, status.Errorf(codes.Internal, "dispatch error: %v", err)
	}

	if driver == nil {
		return &driverv1.MatchDriverResponse{
			Matched: false,
		}, nil
	}

	return &driverv1.MatchDriverResponse{
		Matched:  true,
		DriverId: driver.ID,
		Driver:   mapDomainDriverToProto(driver),
	}, nil
}

// UpdateDriverStatus handles driver status changes (OFFLINE, AVAILABLE, ON_TRIP)
// When status transitions to AVAILABLE, also updates Redis Geo location!
func (h *DriverHandler) UpdateDriverStatus(ctx context.Context, req *driverv1.UpdateDriverStatusRequest) (*driverv1.UpdateDriverStatusResponse, error) {
	if req.DriverId == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}

	statusDomain := mapProtoStatusToDomain(req.Status)

	driver, err := h.repo.GetByID(ctx, req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "driver not found: %v", err)
	}

	if driver.Status != statusDomain {
		if err := driver.ValidateTransition(statusDomain); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		}
	}

	// Update PostgreSQL status
	if err := h.repo.UpdateStatus(ctx, req.DriverId, statusDomain); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update status: %v", err)
	}

	// Update Redis Geo spatial index
	if statusDomain == domain.StatusAvailable {
		if req.Latitude != 0 && req.Longitude != 0 {
			_ = h.geoService.AddDriverLocation(ctx, req.DriverId, req.Latitude, req.Longitude)
		}
	} else {
		_ = h.geoService.RemoveDriver(ctx, req.DriverId)
	}

	driver, err = h.repo.GetByID(ctx, req.DriverId)

	return &driverv1.UpdateDriverStatusResponse{
		Success: true,
		Driver:  mapDomainDriverToProto(driver),
	}, nil
}

// RegisterDriver creates a new driver profile in PostgreSQL
func (h *DriverHandler) RegisterDriver(ctx context.Context, req *driverv1.RegisterDriverRequest) (*driverv1.RegisterDriverResponse, error) {
	if req.Name == "" || req.Phone == "" || req.VehiclePlate == "" {
		return nil, status.Error(codes.InvalidArgument, "name, phone, and vehicle_plate are required")
	}

	driver := &domain.Driver{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		Status:       domain.StatusOffline,
		VehicleType:  req.VehicleType,
		VehiclePlate: req.VehiclePlate,
		VehicleModel: req.VehicleModel,
		Rating:       5.0,
		TotalTrips:   0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if driver.VehicleType == "" {
		driver.VehicleType = "SEDAN"
	}

	if err := h.repo.CreateDriver(ctx, driver); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create driver profile: %v", err)
	}

	return &driverv1.RegisterDriverResponse{
		Driver: mapDomainDriverToProto(driver),
	}, nil
}

// GetDriver retrieves driver profile details by UUID
func (h *DriverHandler) GetDriver(ctx context.Context, req *driverv1.GetDriverRequest) (*driverv1.GetDriverResponse, error) {
	if req.DriverId == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}

	driver, err := h.repo.GetByID(ctx, req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "driver not found: %v", err)
	}

	return &driverv1.GetDriverResponse{
		Driver: mapDomainDriverToProto(driver),
	}, nil
}

// Helper: Maps internal domain.Driver entity -> Protobuf driverv1.Driver message
func mapDomainDriverToProto(d *domain.Driver) *driverv1.Driver {
	statusProto := driverv1.DriverStatus_DRIVER_STATUS_UNSPECIFIED
	switch d.Status {
	case domain.StatusOffline:
		statusProto = driverv1.DriverStatus_DRIVER_STATUS_OFFLINE
	case domain.StatusAvailable:
		statusProto = driverv1.DriverStatus_DRIVER_STATUS_AVAILABLE
	case domain.StatusOnTrip:
		statusProto = driverv1.DriverStatus_DRIVER_STATUS_ON_TRIP
	}

	vTypeProto := driverv1.VehicleType_VEHICLE_TYPE_SEDAN
	switch d.VehicleType {
	case "SEDAN":
		vTypeProto = driverv1.VehicleType_VEHICLE_TYPE_SEDAN
	case "SUV":
		vTypeProto = driverv1.VehicleType_VEHICLE_TYPE_SUV
	case "PREMIUM":
		vTypeProto = driverv1.VehicleType_VEHICLE_TYPE_PREMIUM
	case "BIKE":
		vTypeProto = driverv1.VehicleType_VEHICLE_TYPE_BIKE
	}

	return &driverv1.Driver{
		DriverId:     d.ID,
		Name:         d.Name,
		Phone:        d.Phone,
		Email:        d.Email,
		Status:       statusProto,
		VehicleType:  vTypeProto,
		VehiclePlate: d.VehiclePlate,
		VehicleModel: d.VehicleModel,
		Rating:       d.Rating,
		TotalTrips:   d.TotalTrips,
		CreatedAt:    d.CreatedAt.Unix(),
		UpdatedAt:    d.UpdatedAt.Unix(),
	}
}

// Helper: Maps Protobuf enum status -> internal domain.DriverStatus string type
func mapProtoStatusToDomain(s driverv1.DriverStatus) domain.DriverStatus {
	switch s {
	case driverv1.DriverStatus_DRIVER_STATUS_AVAILABLE:
		return domain.StatusAvailable
	case driverv1.DriverStatus_DRIVER_STATUS_ON_TRIP:
		return domain.StatusOnTrip
	default:
		return domain.StatusOffline
	}
}
