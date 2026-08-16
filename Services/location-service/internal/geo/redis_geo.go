package geo

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// KeyAvailableDrivers is the Redis Geo Sorted Set storing AVAILABLE driver locations
	KeyAvailableDrivers = "drivers:available"
	// KeyOnTripDrivers is the Redis Geo Sorted Set storing ON_TRIP driver locations
	KeyOnTripDrivers = "drivers:on_trip"
	// LocationTTLFormat is used to key per-driver metadata with a TTL
	LocationMetaPrefix = "driver:loc:"
)

// GeoClient handles writing GPS coordinates into Redis Geo spatial indexes.
// The Location Service WRITES here. The Driver Service READS here via GEOSEARCH.
// Both services share the same Redis keyspace — this is by design.
type GeoClient struct {
	client *redis.Client
}

// NewGeoClient creates a new Redis Geo writer for the Location Service
func NewGeoClient(client *redis.Client) *GeoClient {
	return &GeoClient{client: client}
}

// UpdateDriverLocation writes the driver's latest GPS position to Redis Geo.
// If the driver is on an active trip, it updates BOTH the available AND on-trip sets.
// This ensures the Driver Service dispatch loop always sees current positions.
//
// Redis commands executed:
//   - GEOADD drivers:available <lng> <lat> <driver_id>
//   - SET driver:loc:<driver_id> <metadata> EX 30 (30-second TTL for stale detection)
func (g *GeoClient) UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error {
	if g.client == nil {
		return nil // graceful degradation when Redis is offline
	}

	// Always update the available pool — dispatch loop uses this for nearest-driver search.
	// Drivers on a trip are in drivers:on_trip but still tracked for live map display.
	targetKey := KeyAvailableDrivers
	if onTrip {
		targetKey = KeyOnTripDrivers
	}

	if err := g.client.GeoAdd(ctx, targetKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng, // IMPORTANT: Redis Geo expects Longitude FIRST, then Latitude!
		Latitude:  lat,
	}).Err(); err != nil {
		return fmt.Errorf("GEOADD failed for driver %s: %w", driverID, err)
	}

	// Store per-driver metadata with 30-second TTL for stale driver detection.
	// If Location Service stops receiving pings from a driver, this key expires and
	// the system knows the driver went silent (app crash, tunnel, etc.)
	metaKey := fmt.Sprintf("%s%s", LocationMetaPrefix, driverID)
	metaVal := fmt.Sprintf("%.6f,%.6f,%d", lat, lng, time.Now().UnixMilli())
	g.client.Set(ctx, metaKey, metaVal, 30*time.Second)

	return nil
}

// GetDriverLocation fetches the last known position of a specific driver from Redis Geo.
// Uses Redis command: GEOPOS <key> <driver_id>
func (g *GeoClient) GetDriverLocation(ctx context.Context, driverID string) (lat, lng float64, err error) {
	if g.client == nil {
		return 0, 0, fmt.Errorf("redis client not connected")
	}

	// Try available set first, then on-trip set
	for _, key := range []string{KeyAvailableDrivers, KeyOnTripDrivers} {
		positions, err := g.client.GeoPos(ctx, key, driverID).Result()
		if err == nil && len(positions) > 0 && positions[0] != nil {
			return positions[0].Latitude, positions[0].Longitude, nil
		}
	}

	return 0, 0, fmt.Errorf("driver %s not found in any location set", driverID)
}
