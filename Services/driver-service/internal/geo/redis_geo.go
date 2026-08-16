package geo

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// KeyAvailableDrivers is the Redis Sorted Set key storing locations of available drivers
	KeyAvailableDrivers = "drivers:available"
	// KeyOnTripDrivers is the Redis Sorted Set key storing locations of drivers currently on a trip
	KeyOnTripDrivers = "drivers:on_trip"
)

// GeoService handles high-performance in-memory spatial indexing using Redis Geo commands.
// In-memory Redis Geo (`GEOADD`, `GEOSEARCH`) delivers sub-millisecond query performance (~1-2ms),
// making it capable of scaling to 100,000+ concurrent active drivers!
type GeoService struct {
	client *redis.Client // Official go-redis client instance
}

// NewGeoService constructs GeoService instance
func NewGeoService(client *redis.Client) *GeoService {
	return &GeoService{client: client}
}

// AddDriverLocation updates or registers driver coordinates in Redis Geo set `drivers:available`
// Uses Redis command: GEOADD drivers:available <longitude> <latitude> <driver_id>
// NOTE: Redis Geo commands expect Longitude FIRST, then Latitude!
func (g *GeoService) AddDriverLocation(ctx context.Context, driverID string, lat, lng float64) error {
	if g.client == nil {
		return nil
	}

	err := g.client.GeoAdd(ctx, KeyAvailableDrivers, &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng,
		Latitude:  lat,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to GEOADD driver location in Redis: %w", err)
	}

	return nil
}

// RemoveDriver removes a driver from the `drivers:available` geo spatial set (e.g. when going offline or assigned to trip)
// Uses Redis command: ZREM drivers:available <driver_id>
func (g *GeoService) RemoveDriver(ctx context.Context, driverID string) error {
	if g.client == nil {
		return nil
	}

	err := g.client.ZRem(ctx, KeyAvailableDrivers, driverID).Err()
	if err != nil {
		return fmt.Errorf("failed to ZREM driver from Redis available set: %w", err)
	}

	return nil
}

// FindNearbyDrivers executes sub-millisecond GEOSEARCH query to find nearest available drivers within a radius
// Uses Redis command: GEOSEARCH drivers:available FROMLONLAT <lng> <lat> BYRADIUS <radiusKm> km ASC COUNT <limit>
func (g *GeoService) FindNearbyDrivers(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
	if g.client == nil {
		return nil, nil
	}

	query := &redis.GeoSearchQuery{
		Longitude:  lng,
		Latitude:   lat,
		Radius:     radiusKm,
		RadiusUnit: "km",
		Sort:       "ASC", // Sort closest drivers first!
		Count:      limit,
	}

	locations, err := g.client.GeoSearch(ctx, KeyAvailableDrivers, query).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to GEOSEARCH nearby drivers: %w", err)
	}

	results := make([]redis.GeoLocation, 0, len(locations))
	for _, locName := range locations {
		results = append(results, redis.GeoLocation{
			Name: locName,
		})
	}

	return results, nil
}

// AcquireDispatchLock attempts to acquire a short-lived atomic distributed lock for offering a ride to a driver
// Uses Redis command: SET lock:driver:dispatch:<driver_id> <trip_id> NX PX 15000
// WHY DISTRIBUTED LOCKING IS CRITICAL:
// If two riders request a trip at the exact same second in the same area, without distributed locking,
// the system might offer the same driver to both riders simultaneously!
// Redis `SetNX` (Set if Not Exists) acts as an atomic lock, ensuring only one dispatch offer touches a driver at a time.
func (g *GeoService) AcquireDispatchLock(ctx context.Context, driverID, tripID string, ttl time.Duration) (bool, error) {
	if g.client == nil {
		return true, nil // Fallback mock lock if Redis is unconfigured
	}

	lockKey := fmt.Sprintf("lock:driver:dispatch:%s", driverID)
	acquired, err := g.client.SetNX(ctx, lockKey, tripID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire driver dispatch lock: %w", err)
	}

	return acquired, nil
}

// ReleaseDispatchLock releases the atomic distributed lock for a driver after dispatch completes or times out
func (g *GeoService) ReleaseDispatchLock(ctx context.Context, driverID string) error {
	if g.client == nil {
		return nil
	}

	lockKey := fmt.Sprintf("lock:driver:dispatch:%s", driverID)
	_ = g.client.Del(ctx, lockKey).Err()
	return nil
}
