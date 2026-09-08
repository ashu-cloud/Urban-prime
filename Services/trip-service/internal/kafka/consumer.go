package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	// TopicMatchEvents is the Redis Stream key that the Driver Service publishes
	// ACCEPTED / EXHAUSTED match outcomes to. Trip Service consumes this to
	// advance the Saga state machine.
	TopicMatchEvents = "driver.match.v1"
)

type matchEventMessage struct {
	EventType string `json:"event_type"` // ACCEPTED, DECLINED, EXHAUSTED
	TripID    string `json:"trip_id"`
	DriverID  string `json:"driver_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type AssignDriverFunc func(ctx context.Context, tripID, driverID string) error
type CompensateNoDriverFunc func(ctx context.Context, tripID string)

// Consumer reads from the Redis Stream "driver.match.v1" and advances the
// Trip Saga state machine upon driver acceptance or exhaustion.
type Consumer struct {
	client                 *redis.Client
	groupID                string
	consumerName           string
	assignDriverFunc       AssignDriverFunc
	compensateNoDriverFunc CompensateNoDriverFunc
}

func NewConsumer(redisAddr, groupID string, assignDriverFunc AssignDriverFunc, compensateNoDriverFunc CompensateNoDriverFunc) (*Consumer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for trip consumer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)

	c := &Consumer{
		client:                 client,
		groupID:                groupID,
		consumerName:           "trip-service-1",
		assignDriverFunc:       assignDriverFunc,
		compensateNoDriverFunc: compensateNoDriverFunc,
	}

	// Ensure the consumer group exists. $ means: only deliver new messages.
	// MKSTREAM creates the stream if it does not yet exist.
	bgCtx := context.Background()
	if err := client.XGroupCreateMkStream(bgCtx, TopicMatchEvents, groupID, "$").Err(); err != nil {
		// BUSYGROUP means the group already exists — perfectly normal.
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			// Non-fatal: log and continue — first messages may be missed but service recovers
			logger.Warn(bgCtx, "Trip consumer group create warning", "stream", TopicMatchEvents, "error", err)
		}
	}

	return c, nil
}

// Start begins a blocking Redis XREADGROUP loop.
// Runs as a goroutine — returns when ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	logger.Info(ctx, "Trip Service Redis stream consumer started",
		"stream", TopicMatchEvents,
		"group", c.groupID,
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Trip Service stream consumer stopping...")
			_ = c.client.Close()
			return
		default:
		}

		// Block up to 2 seconds waiting for new messages
		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.groupID,
			Consumer: c.consumerName,
			Streams:  []string{TopicMatchEvents, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()

		if err != nil {
			// redis.Nil means timeout (no messages) — normal, just loop again
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return // context cancelled
			}
			logger.Warn(ctx, "Trip stream XREADGROUP error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.handleMessage(ctx, msg)
				// ACK the message so it is not re-delivered
				_ = c.client.XAck(ctx, TopicMatchEvents, c.groupID, msg.ID).Err()
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg redis.XMessage) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		logger.Warn(ctx, "Trip consumer: message missing payload field", "id", msg.ID)
		return
	}

	var event matchEventMessage
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		logger.Warn(ctx, "Trip consumer: failed to unmarshal match event", "error", err)
		return
	}

	switch event.EventType {
	case "ACCEPTED":
		logger.Info(ctx, "Saga Step 3 Completed: Driver ACCEPTED trip",
			"trip_id", event.TripID,
			"driver_id", event.DriverID,
		)
		if c.assignDriverFunc != nil {
			if err := c.assignDriverFunc(ctx, event.TripID, event.DriverID); err != nil {
				logger.Error(ctx, "Failed to assign driver in Saga Orchestrator",
					"trip_id", event.TripID,
					"error", err,
				)
			}
		}

	case "EXHAUSTED":
		logger.Warn(ctx, "Saga Compensation Triggered: All candidate drivers exhausted",
			"trip_id", event.TripID,
		)
		if c.compensateNoDriverFunc != nil {
			c.compensateNoDriverFunc(ctx, event.TripID)
		}
	}
}
