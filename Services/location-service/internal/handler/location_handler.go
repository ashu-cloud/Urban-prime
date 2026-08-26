package handler

import (
	"context"
	"time"

	locationv1 "github.com/cab-booking/proto/gen/location/v1"
	"github.com/cab-booking/location-service/internal/kafka"
	"github.com/cab-booking/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GeoClientInterface interface {
	UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error
	GetDriverLocation(ctx context.Context, driverID string) (float64, float64, error)
}

type KafkaProducerInterface interface {
	PublishLocationUpdate(ctx context.Context, event kafka.LocationEvent) error
}

// LocationHandler implements the locationv1.LocationServiceServer gRPC interface.
// This is the hottest code path in the entire platform — every GPS ping from every
// active driver hits this handler every 3 seconds!
type LocationHandler struct {
	locationv1.UnimplementedLocationServiceServer
	geoClient GeoClientInterface    // Writes to Redis Geo spatial index
	producer  KafkaProducerInterface   // Publishes to Kafka topic `driver.location.v1`
}

// NewLocationHandler constructs the gRPC handler
func NewLocationHandler(geoClient GeoClientInterface, producer KafkaProducerInterface) *LocationHandler {
	return &LocationHandler{
		geoClient: geoClient,
		producer:  producer,
	}
}

// UpdateDriverLocation is called by the driver's mobile app on every GPS ping.
//
// What happens inside (in order):
//  1. Validate the request (driver_id and coordinates are required)
//  2. Write GPS coordinates to Redis Geo → instantly available for dispatch matching
//  3. Publish `LocationUpdated` event to Kafka → Notification Service reads this
//     and forwards the position to the rider's map via Centrifugo WebSocket
//
// Design note: Redis write is synchronous (we need it immediately for dispatch).
// Kafka publish is best-effort (non-fatal if it fails — map update just misses one frame).
func (h *LocationHandler) UpdateDriverLocation(ctx context.Context, req *locationv1.UpdateDriverLocationRequest) (*locationv1.UpdateDriverLocationResponse, error) {
	// Input validation
	if req.DriverId == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}
	if req.Latitude == 0 && req.Longitude == 0 {
		return nil, status.Error(codes.InvalidArgument, "latitude and longitude are required")
	}

	onTrip := req.TripId != ""

	// STEP 1: Write to Redis Geo (synchronous — dispatch loop needs this immediately)
	if err := h.geoClient.UpdateDriverLocation(ctx, req.DriverId, req.Latitude, req.Longitude, onTrip); err != nil {
		logger.Error(ctx, "Failed to write driver location to Redis Geo",
			"driver_id", req.DriverId,
			"error", err,
		)
		// We still continue to publish Kafka event even if Redis fails
	}

	// STEP 2: Publish to Kafka (async best-effort — for downstream WebSocket fanout)
	// The Notification Service consumes this event and pushes to the rider's Centrifugo channel
	_ = h.producer.PublishLocationUpdate(ctx, kafka.LocationEvent{
		DriverID:  req.DriverId,
		TripID:    req.TripId,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		SpeedKmh:  req.SpeedKmh,
		Bearing:   req.Bearing,
		Timestamp: time.Now(),
	})

	return &locationv1.UpdateDriverLocationResponse{Success: true}, nil
}

// GetDriverLocation returns the last known GPS position of a driver.
// Used by the rider app to show initial driver position when a trip is matched.
func (h *LocationHandler) GetDriverLocation(ctx context.Context, req *locationv1.GetDriverLocationRequest) (*locationv1.GetDriverLocationResponse, error) {
	if req.DriverId == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}

	lat, lng, err := h.geoClient.GetDriverLocation(ctx, req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "driver location not found: %v", err)
	}

	return &locationv1.GetDriverLocationResponse{
		DriverId:  req.DriverId,
		Latitude:  lat,
		Longitude: lng,
		UpdatedAt: time.Now().Unix(),
	}, nil
}
