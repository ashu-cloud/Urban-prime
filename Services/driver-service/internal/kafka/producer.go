package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// MatchEventPayload represents match dispatch lifecycle events published to
// the Redis Stream "driver.match.v1".
type MatchEventPayload struct {
	EventType string    `json:"event_type"` // "OFFERED", "ACCEPTED", "DECLINED", "EXHAUSTED"
	TripID    string    `json:"trip_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Attempts  int       `json:"attempts,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Producer publishes driver match lifecycle events to a Redis Stream.
type Producer struct {
	client *redis.Client
}

func NewProducer(redisAddr string) (*Producer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for driver producer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)
	return &Producer{client: client}, nil
}

// PublishMatchEvent appends a driver match lifecycle event to the Redis Stream
// identified by `stream` (e.g. "driver.match.v1").
func (p *Producer) PublishMatchEvent(ctx context.Context, stream string, payload MatchEventPayload) error {
	if p.client == nil {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal match event payload: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := &redis.XAddArgs{
		Stream: stream,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{
			"payload": string(data),
		},
	}

	if err := p.client.XAdd(ctxTimeout, args).Err(); err != nil {
		logger.Warn(ctx, "Redis stream XADD failed for match event",
			"stream", stream,
			"trip_id", payload.TripID,
			"event", payload.EventType,
			"error", err,
		)
	} else {
		logger.Info(ctx, "Published match event to Redis stream",
			"stream", stream,
			"trip_id", payload.TripID,
			"event", payload.EventType,
		)
	}

	return nil
}

func (p *Producer) Close() {
	if p.client != nil {
		_ = p.client.Close()
	}
}
