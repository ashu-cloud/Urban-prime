package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// TopicDriverLocation is the Redis Stream key that carries all driver GPS update events.
// The Notification Service consumes from this stream to push real-time map updates.
const TopicDriverLocation = "driver.location.v1"

// LocationEvent is the payload published for every GPS ping from a driver.
// Downstream consumers (Notification Service) read this to push live map updates to riders.
type LocationEvent struct {
	DriverID  string    `json:"driver_id"`
	TripID    string    `json:"trip_id,omitempty"` // empty if driver is not currently on a trip
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	SpeedKmh  float32   `json:"speed_kmh,omitempty"`
	Bearing   float32   `json:"bearing,omitempty"` // compass direction 0-360 degrees
	Timestamp time.Time `json:"timestamp"`
}

// Producer publishes LocationEvent messages to a Redis Stream.
type Producer struct {
	client *redis.Client
}

// NewProducer creates a Redis-backed stream producer for the Location Service.
func NewProducer(redisAddr string) (*Producer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for location producer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)
	return &Producer{client: client}, nil
}

// PublishLocationUpdate appends a driver GPS update event to the Redis Stream.
// Key = DriverID is embedded in the payload; MAXLEN ~ 50000 keeps ~last 50k GPS pings.
// Non-fatal if it fails — Redis Geo is already updated with the new position.
func (p *Producer) PublishLocationUpdate(ctx context.Context, event LocationEvent) error {
	if p.client == nil {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal location event: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := &redis.XAddArgs{
		Stream: TopicDriverLocation,
		MaxLen: 50000,
		Approx: true,
		Values: map[string]interface{}{
			"payload": string(data),
		},
	}

	if err := p.client.XAdd(ctxTimeout, args).Err(); err != nil {
		// Non-fatal: location data is already written to Redis Geo. Stream is for downstream fanout.
		logger.Warn(ctx, "Redis stream location event publish failed (Redis Geo already updated)",
			"driver_id", event.DriverID,
			"error", err,
		)
	}

	return nil
}

func (p *Producer) Close() {
	if p.client != nil {
		_ = p.client.Close()
	}
}
