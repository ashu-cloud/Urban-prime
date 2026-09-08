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
	// TopicTripEvents is the Redis Stream key that the Trip Service publishes
	// trip lifecycle events to. Driver Service consumes this to trigger dispatch.
	TopicTripEvents = "trip.events.v1"
)

type tripEventMessage struct {
	TripID      string  `json:"trip_id"`
	Status      string  `json:"status"`
	PickupLat   float64 `json:"pickup_lat"`
	PickupLng   float64 `json:"pickup_lng"`
	VehicleType string  `json:"vehicle_type,omitempty"`
}

// DispatchFunc is a function signature for triggering driver matchmaking
type DispatchFunc func(ctx context.Context, tripID string, pickupLat, pickupLng float64, vehicleType string) error

// Consumer reads from the Redis Stream "trip.events.v1" and triggers
// the Driver Service dispatch loop when a trip enters MATCHING state.
type Consumer struct {
	client       *redis.Client
	groupID      string
	consumerName string
	dispatchFunc DispatchFunc
}

func NewConsumer(redisAddr, groupID string, dispatchFunc DispatchFunc) (*Consumer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for driver consumer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)

	c := &Consumer{
		client:       client,
		groupID:      groupID,
		consumerName: "driver-service-1",
		dispatchFunc: dispatchFunc,
	}

	// Create consumer group if it does not yet exist
	bgCtx := context.Background()
	if err := client.XGroupCreateMkStream(bgCtx, TopicTripEvents, groupID, "$").Err(); err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			logger.Warn(bgCtx, "Driver consumer group create warning", "stream", TopicTripEvents, "error", err)
		}
	}

	return c, nil
}

// Start begins a blocking Redis XREADGROUP loop.
// Runs as a goroutine — returns when ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	logger.Info(ctx, "Driver Service Redis stream consumer started — listening for trip MATCHING events",
		"stream", TopicTripEvents,
		"group", c.groupID,
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Driver Service stream consumer stopping...")
			_ = c.client.Close()
			return
		default:
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.groupID,
			Consumer: c.consumerName,
			Streams:  []string{TopicTripEvents, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			logger.Warn(ctx, "Driver stream XREADGROUP error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.handleMessage(ctx, msg)
				_ = c.client.XAck(ctx, TopicTripEvents, c.groupID, msg.ID).Err()
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg redis.XMessage) {
	raw, ok := msg.Values["payload"].(string)
	if !ok {
		logger.Warn(ctx, "Driver consumer: message missing payload field", "id", msg.ID)
		return
	}

	var event tripEventMessage
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		logger.Warn(ctx, "Driver consumer: failed to unmarshal trip event", "error", err)
		return
	}

	// WHEN A TRIP ENTERS 'MATCHING' STATUS → TRIGGER DISPATCH LOOP!
	if event.Status == "MATCHING" && c.dispatchFunc != nil {
		logger.Info(ctx, "Stream Event Received: Trip in MATCHING state — triggering Dispatch Loop",
			"trip_id", event.TripID,
		)
		go func(e tripEventMessage) {
			dispatchCtx := context.Background()
			if err := c.dispatchFunc(dispatchCtx, e.TripID, e.PickupLat, e.PickupLng, e.VehicleType); err != nil {
				logger.Error(dispatchCtx, "Dispatch loop execution returned error",
					"trip_id", e.TripID,
					"error", err,
				)
			}
		}(event)
	}
}
