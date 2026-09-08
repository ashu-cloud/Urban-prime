package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// TripEventPayload is the event published to Redis Stream "trip.events.v1"
// for every significant trip lifecycle state transition.
type TripEventPayload struct {
	TripID        string    `json:"trip_id"`
	RiderID       string    `json:"rider_id"`
	Status        string    `json:"status"`
	PickupLat     float64   `json:"pickup_lat"`
	PickupLng     float64   `json:"pickup_lng"`
	DropoffLat    float64   `json:"dropoff_lat"`
	DropoffLng    float64   `json:"dropoff_lng"`
	VehicleType   string    `json:"vehicle_type"`
	DistanceKm    float64   `json:"distance_km"`
	EstimatedFare int64     `json:"estimated_fare_cents"`
	Currency      string    `json:"currency"`
	Timestamp     time.Time `json:"timestamp"`
}

// Producer publishes trip lifecycle events to a Redis Stream.
type Producer struct {
	client *redis.Client
}

// NewProducer creates a Redis-backed stream producer.
// redisAddr is the Redis server address (e.g. "localhost:6379" or a rediss:// URL).
func NewProducer(redisAddr string) (*Producer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for trip producer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)
	return &Producer{client: client}, nil
}

// PublishTripEvent appends a trip event to the Redis Stream identified by `stream`.
// The stream name maps 1:1 to what was previously a Kafka topic name ("trip.events.v1").
// Uses MAXLEN ~ 10000 so the stream does not grow unboundedly.
func (p *Producer) PublishTripEvent(ctx context.Context, stream string, payload TripEventPayload) error {
	if p.client == nil {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal trip event payload: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := &redis.XAddArgs{
		Stream: stream,
		MaxLen: 10000,
		Approx: true, // MAXLEN ~ (approximate, much faster)
		Values: map[string]interface{}{
			"payload": string(data),
		},
	}

	if err := p.client.XAdd(ctxTimeout, args).Err(); err != nil {
		logger.Warn(ctx, "Redis stream XADD failed, event lost",
			"stream", stream,
			"trip_id", payload.TripID,
			"status", payload.Status,
			"error", err,
		)
	} else {
		logger.Info(ctx, "Published trip event to Redis stream",
			"stream", stream,
			"trip_id", payload.TripID,
			"status", payload.Status,
		)
	}

	return nil
}

// Close closes the underlying Redis client.
func (p *Producer) Close() {
	if p.client != nil {
		_ = p.client.Close()
	}
}
