package dispatch

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/cab-booking/driver-service/internal/domain"
	"github.com/cab-booking/driver-service/internal/kafka"
	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultSearchRadiusKm       = 50000.0 // Global search radius for local testing
	DefaultMaxCandidates        = 10      // Maximum number of nearby candidates to pull from Redis GEOSEARCH
	DefaultMaxOffers            = 5       // Maximum driver dispatch attempts before declaring match exhausted
	DefaultOfferTimeoutSecs     = 15      // Wait window for driver to accept/decline ride offer (15 seconds)
	DefaultDispatchWindowSecs   = 20      // Overall matching window duration (matching UI timer)
	DefaultPollingIntervalSecs = 2       // Interval between spatial searches for newly available drivers
)

// DispatchLoop is the core matchmaking engine for ride allocation.
// It implements the dispatch loop algorithm:
// 1. Continually query Redis GEOSEARCH within the matching window for closest active drivers.
// 2. Filter candidates by requested vehicle type.
// 3. Acquire an atomic Redis distributed lock (`SetNX`) per driver to avoid race conditions.
// 4. Offer ride sequentially (closest driver first).
// 5. Emit Kafka `MatchOffered` event.
// 6. Handle acceptance (transition to ON_TRIP, remove from available set) or decline/timeout (release lock, try next nearest candidate).
type GeoServiceInterface interface {
	FindNearbyDrivers(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error)
	AcquireDispatchLock(ctx context.Context, driverID, tripID string, ttl time.Duration) (bool, error)
	ReleaseDispatchLock(ctx context.Context, driverID string) error
	RemoveDriver(ctx context.Context, driverID string) error
	WaitForDriverResponse(ctx context.Context, driverID, tripID string, timeout time.Duration) bool
	PublishDriverResponse(ctx context.Context, driverID, tripID string, accepted bool) error
}

type DriverRepoInterface interface {
	GetByID(ctx context.Context, id string) (*domain.Driver, error)
	UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error
}

type KafkaProducerInterface interface {
	PublishMatchEvent(ctx context.Context, topic string, payload kafka.MatchEventPayload) error
}

// DispatchLoop is the core matchmaking engine for ride allocation.
type DispatchLoop struct {
	geoService             GeoServiceInterface
	repo                   DriverRepoInterface
	producer               KafkaProducerInterface
	SimulateDriverResponse func(ctx context.Context, driverID string) bool
	MaxDispatchDuration    time.Duration
	PollingInterval        time.Duration
}

// NewDispatchLoop constructs a new DispatchLoop instance
func NewDispatchLoop(geoService GeoServiceInterface, repo DriverRepoInterface, producer KafkaProducerInterface) *DispatchLoop {
	return &DispatchLoop{
		geoService:             geoService,
		repo:                   repo,
		producer:               producer,
		SimulateDriverResponse: nil,
		MaxDispatchDuration:    DefaultDispatchWindowSecs * time.Second,
		PollingInterval:        DefaultPollingIntervalSecs * time.Second,
	}
}

// FindAndDispatchDriver executes the nearest-driver matchmaking algorithm over a configurable matching window
func (d *DispatchLoop) FindAndDispatchDriver(
	ctx context.Context,
	tripID string,
	pickupLat float64,
	pickupLng float64,
	requestedVehicleType string,
) (*domain.Driver, error) {
	logger.Info(ctx, "Dispatch Loop Initiated: Searching nearest available drivers via Redis Geo",
		"trip_id", tripID,
		"lat", pickupLat,
		"lng", pickupLng,
		"vehicle_type", requestedVehicleType,
		"max_duration", d.MaxDispatchDuration,
	)

	seenDrivers := make(map[string]bool)
	attempts := 0
	startTime := time.Now()

	for time.Since(startTime) < d.MaxDispatchDuration {
		select {
		case <-ctx.Done():
			logger.Warn(ctx, "Dispatch loop cancelled by context", "trip_id", tripID, "error", ctx.Err())
			return nil, ctx.Err()
		default:
		}

		if attempts >= DefaultMaxOffers {
			logger.Warn(ctx, "Max dispatch offer attempts reached", "max", DefaultMaxOffers, "trip_id", tripID)
			break
		}

		// STEP 1: EXECUTE Redis GEOSEARCH QUERY
		candidates, err := d.geoService.FindNearbyDrivers(ctx, pickupLat, pickupLng, DefaultSearchRadiusKm, DefaultMaxCandidates)
		if err != nil {
			logger.Error(ctx, "Redis GEOSEARCH failed during dispatch loop", "error", err)
		}

		// STEP 2: ITERATE CANDIDATES SEQUENTIALLY (Closest driver first)
		for _, candidate := range candidates {
			driverID := candidate.Name
			if driverID == "" || seenDrivers[driverID] {
				continue
			}

			if attempts >= DefaultMaxOffers {
				break
			}

			// Mark seen so we do not spam the same driver if they declined or timed out
			seenDrivers[driverID] = true
			attempts++

			// Verify driver profile & vehicle type match in PostgreSQL database
			driver, err := d.repo.GetByID(ctx, driverID)
			if err != nil {
				logger.Warn(ctx, "Skipping candidate driver: profile not found in DB", "driver_id", driverID, "error", err)
				continue
			}

			// Ensure driver is currently AVAILABLE
			if driver.Status != domain.StatusAvailable {
				logger.Debug(ctx, "Skipping candidate driver: status not AVAILABLE", "driver_id", driverID, "status", driver.Status)
				continue
			}

			// Filter by requested vehicle type (e.g., SEDAN, SUV, PREMIUM, BIKE)
			if requestedVehicleType != "" && driver.VehicleType != requestedVehicleType {
				logger.Debug(ctx, "Skipping candidate driver: vehicle type mismatch", "driver_id", driverID, "driver_type", driver.VehicleType, "requested", requestedVehicleType)
				continue
			}

			// STEP 3a: ACQUIRE ATOMIC DISTRIBUTED LOCK IN REDIS
			// Prevents race conditions where two riders are offered the exact same driver simultaneously!
			locked, err := d.geoService.AcquireDispatchLock(ctx, driverID, tripID, DefaultOfferTimeoutSecs*time.Second)
			if err != nil || !locked {
				logger.Warn(ctx, "Skipping candidate driver: dispatch lock acquisition failed (driver currently receiving another offer)", "driver_id", driverID)
				continue
			}

			// STEP 3b: PUBLISH `MatchOffered` EVENT TO KAFKA / REDIS STREAM TOPIC `driver.match.v1`
			if err := d.producer.PublishMatchEvent(ctx, "driver.match.v1", kafka.MatchEventPayload{
				EventType: "OFFERED",
				TripID:    tripID,
				DriverID:  driverID,
				Timestamp: time.Now(),
			}); err != nil {
				logger.Warn(ctx, "Failed to publish OFFERED match event", "error", err)
			}

			logger.Info(ctx, fmt.Sprintf("Dispatch Offer #%d sent to driver", attempts), "driver_id", driverID, "driver_name", driver.Name, "trip_id", tripID)

			// STEP 3c: OFFER RESPONSE HANDLING / ACCEPTANCE WINDOW
			var accepted bool
			if d.SimulateDriverResponse != nil {
				accepted = d.SimulateDriverResponse(ctx, driverID)
			} else {
				accepted = d.geoService.WaitForDriverResponse(ctx, driverID, tripID, DefaultOfferTimeoutSecs*time.Second)
			}

			if accepted {
				logger.Info(ctx, "Driver ACCEPTED ride offer!", "driver_id", driverID, "trip_id", tripID)

				// Update driver status in PostgreSQL to ON_TRIP
				if err := d.repo.UpdateStatus(ctx, driverID, domain.StatusOnTrip); err != nil {
					logger.Error(ctx, "Failed to update driver status to ON_TRIP in DB", "driver_id", driverID, "error", err)
				}

				// Remove driver from Redis `drivers:available` geo spatial set
				_ = d.geoService.RemoveDriver(ctx, driverID)

				// Release atomic dispatch lock
				_ = d.geoService.ReleaseDispatchLock(ctx, driverID)

				// Publish `MatchAccepted` event to Kafka
				if err := d.producer.PublishMatchEvent(ctx, "driver.match.v1", kafka.MatchEventPayload{
					EventType: "ACCEPTED",
					TripID:    tripID,
					DriverID:  driverID,
					Timestamp: time.Now(),
				}); err != nil {
					logger.Warn(ctx, "Failed to publish ACCEPTED match event", "error", err)
				}

				driver.Status = domain.StatusOnTrip
				return driver, nil
			}

			// STEP 3d: DRIVER DECLINED OR TIMED OUT
			logger.Warn(ctx, "Driver DECLINED or timed out on ride offer. Releasing lock & trying next nearest driver...", "driver_id", driverID, "trip_id", tripID)

			// Release distributed lock so driver can receive future trip offers
			_ = d.geoService.ReleaseDispatchLock(ctx, driverID)

			// Publish `MatchDeclined` event to Kafka
			if err := d.producer.PublishMatchEvent(ctx, "driver.match.v1", kafka.MatchEventPayload{
				EventType: "DECLINED",
				TripID:    tripID,
				DriverID:  driverID,
				Reason:    "Timeout or declined by driver",
				Timestamp: time.Now(),
			}); err != nil {
				logger.Warn(ctx, "Failed to publish DECLINED match event", "error", err)
			}
		}

		if attempts >= DefaultMaxOffers {
			break
		}

		// If time remains in the matching window, wait before polling for newly available drivers
		remaining := d.MaxDispatchDuration - time.Since(startTime)
		if remaining <= 0 {
			break
		}

		sleepDuration := d.PollingInterval
		if sleepDuration > remaining {
			sleepDuration = remaining
		}

		select {
		case <-ctx.Done():
			logger.Warn(ctx, "Dispatch loop cancelled while waiting for drivers", "trip_id", tripID)
			return nil, ctx.Err()
		case <-time.After(sleepDuration):
		}
	}

	// STEP 4: ALL CANDIDATES EXHAUSTED WITHOUT MATCH ACCEPTANCE
	logger.Warn(ctx, "Dispatch Loop Exhausted: No available drivers accepted the trip request", "trip_id", tripID, "attempts", attempts)

	if err := d.producer.PublishMatchEvent(ctx, "driver.match.v1", kafka.MatchEventPayload{
		EventType: "EXHAUSTED",
		TripID:    tripID,
		Attempts:  attempts,
		Timestamp: time.Now(),
	}); err != nil {
		logger.Warn(ctx, "Failed to publish EXHAUSTED match event", "error", err)
	}

	return nil, nil
}

// DefaultSimulateDriverResponse simulates a realistic driver acceptance window for testing.
// 70% acceptance rate with a 1-second simulated think-time.
func DefaultSimulateDriverResponse(ctx context.Context, _ string) bool {
	// Simulate driver "thinking" about the offer
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		return false
	}
	// 70% acceptance rate — realistic for a busy platform
	return rand.Float32() < 0.70
}
