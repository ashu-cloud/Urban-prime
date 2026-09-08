package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cab-booking/notification-service/internal/centrifugo"
	"github.com/cab-booking/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Redis Stream keys consumed by the Notification Service
const (
	TopicDriverLocation = "driver.location.v1" // GPS pings from active drivers
	TopicTripEvents     = "trip.events.v1"      // Trip lifecycle events (created, matched, completed)
	TopicMatchEvents    = "driver.match.v1"     // Driver dispatch events (offered, accepted, declined)
)

// locationEvent mirrors the payload published by the Location Service
type locationEvent struct {
	DriverID  string    `json:"driver_id"`
	TripID    string    `json:"trip_id,omitempty"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	SpeedKmh  float32   `json:"speed_kmh,omitempty"`
	Bearing   float32   `json:"bearing,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// tripEvent mirrors the payload published by the Trip Service
type tripEvent struct {
	TripID        string    `json:"trip_id"`
	RiderID       string    `json:"rider_id"`
	DriverID      string    `json:"driver_id,omitempty"`
	Status        string    `json:"status"`
	PickupLat     float64   `json:"pickup_lat,omitempty"`
	PickupLng     float64   `json:"pickup_lng,omitempty"`
	EstimatedFare int64     `json:"estimated_fare_cents,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// matchEvent mirrors the payload published by the Driver Service
type matchEvent struct {
	EventType string    `json:"event_type"` // OFFERED, ACCEPTED, DECLINED, EXHAUSTED
	TripID    string    `json:"trip_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// KafkaConsumer is the core of the Notification Service.
// (Name kept as KafkaConsumer for zero-change compatibility in main.go and tests.)
// It reads events from Redis Streams and translates them into Centrifugo WebSocket broadcasts.
//
// HOW IT SCALES:
// Redis consumer groups allow multiple instances of the Notification Service to run in parallel.
// Redis automatically delivers unACKed messages to available group members — no duplicate processing.
type KafkaConsumer struct {
	client       *redis.Client
	groupID      string
	consumerName string
	centrifugo   *centrifugo.Client
}

// NewKafkaConsumer creates a Redis Streams consumer that reads from all three event streams.
func NewKafkaConsumer(redisAddr, groupID string, centrifugoClient *centrifugo.Client) (*KafkaConsumer, error) {
	var opt *redis.Options
	var err error

	if len(redisAddr) > 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = redis.ParseURL(redisAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL for notification consumer: %w", err)
		}
	} else {
		opt = &redis.Options{Addr: redisAddr}
	}

	opt.DialTimeout = 3 * time.Second
	client := redis.NewClient(opt)

	c := &KafkaConsumer{
		client:       client,
		groupID:      groupID,
		consumerName: "notification-service-1",
		centrifugo:   centrifugoClient,
	}

	// Ensure consumer groups exist for all three streams
	bgCtx := context.Background()
	for _, stream := range []string{TopicDriverLocation, TopicTripEvents, TopicMatchEvents} {
		if err := client.XGroupCreateMkStream(bgCtx, stream, groupID, "$").Err(); err != nil {
			if err.Error() != "BUSYGROUP Consumer Group name already exists" {
				logger.Warn(bgCtx, "Notification consumer group create warning", "stream", stream, "error", err)
			}
		}
	}

	return c, nil
}

// Start begins the continuous event consumption loop.
// This runs as a goroutine in main.go and processes events indefinitely until ctx is cancelled.
//
// The loop:
//  1. XREADGROUP across all 3 streams in a single blocking call (~100ms timeout)
//  2. For each record, dispatch to the appropriate handler by stream name
//  3. XACKs each message so it is not re-delivered to another consumer in the group
func (k *KafkaConsumer) Start(ctx context.Context) {
	logger.Info(ctx, "Redis stream consumer started — listening for real-time events",
		"streams", []string{TopicDriverLocation, TopicTripEvents, TopicMatchEvents},
		"group", k.groupID,
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Redis stream consumer stopping — context cancelled")
			_ = k.client.Close()
			return
		default:
		}

		// XREADGROUP blocks up to 100ms waiting for new messages across all 3 streams
		streams, err := k.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    k.groupID,
			Consumer: k.consumerName,
			Streams:  []string{TopicDriverLocation, TopicTripEvents, TopicMatchEvents, ">", ">", ">"},
			Count:    50,
			Block:    100 * time.Millisecond,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue // timeout — no messages, loop again
			}
			if ctx.Err() != nil {
				return // context cancelled
			}
			logger.Warn(ctx, "Redis stream XREADGROUP error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				switch stream.Stream {
				case TopicDriverLocation:
					k.handleLocationEvent(ctx, msg.Values)
				case TopicTripEvents:
					k.handleTripEvent(ctx, msg.Values)
				case TopicMatchEvents:
					k.handleMatchEvent(ctx, msg.Values)
				}
				// ACK the message so it is not re-delivered
				_ = k.client.XAck(ctx, stream.Stream, k.groupID, msg.ID).Err()
			}
		}
	}
}

// handleLocationEvent translates a Redis stream location update into a Centrifugo WebSocket push.
// This is the critical path for the "driver moving on map" live-tracking feature!
func (k *KafkaConsumer) handleLocationEvent(ctx context.Context, values map[string]interface{}) {
	raw, ok := values["payload"].(string)
	if !ok {
		return
	}
	var event locationEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		logger.Warn(ctx, "Failed to parse location event", "error", err)
		return
	}

	// Only push live tracking if the driver is on an active trip
	if event.TripID != "" {
		k.centrifugo.PublishDriverLocation(ctx, event.TripID, event.DriverID, event.Latitude, event.Longitude, event.Bearing)
	}
}

// handleTripEvent translates trip lifecycle events into Centrifugo WebSocket pushes.
// Rider app receives: "Driver matched!", "Trip started!", "Trip completed!" notifications.
func (k *KafkaConsumer) handleTripEvent(ctx context.Context, values map[string]interface{}) {
	raw, ok := values["payload"].(string)
	if !ok {
		return
	}
	var event tripEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		logger.Warn(ctx, "Failed to parse trip event", "error", err)
		return
	}

	payload := map[string]interface{}{
		"trip_id":   event.TripID,
		"rider_id":  event.RiderID,
		"driver_id": event.DriverID,
		"status":    event.Status,
	}

	eventType := fmt.Sprintf("TRIP_%s", event.Status)
	k.centrifugo.PublishTripEvent(ctx, event.TripID, eventType, payload)

	logger.Info(ctx, "Pushed trip event to Centrifugo channel",
		"trip_id", event.TripID,
		"status", event.Status,
	)
}

// handleMatchEvent handles driver dispatch events (ACCEPTED, DECLINED, EXHAUSTED).
// Rider app receives: "Driver is on the way!" or "No drivers available, please try again."
func (k *KafkaConsumer) handleMatchEvent(ctx context.Context, values map[string]interface{}) {
	raw, ok := values["payload"].(string)
	if !ok {
		return
	}
	var event matchEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		logger.Warn(ctx, "Failed to parse match event", "error", err)
		return
	}

	if event.TripID == "" {
		return
	}

	payload := map[string]interface{}{
		"trip_id":   event.TripID,
		"driver_id": event.DriverID,
	}

	k.centrifugo.PublishTripEvent(ctx, event.TripID, fmt.Sprintf("MATCH_%s", event.EventType), payload)

	// Direct real-time dispatch prompt to the specific driver's channel
	if event.EventType == "OFFERED" && event.DriverID != "" {
		driverChannel := fmt.Sprintf("driver#%s", event.DriverID)
		dispatchOfferPayload := map[string]interface{}{
			"type":               "DISPATCH_OFFER",
			"event_type":         "DISPATCH_OFFER",
			"trip_id":            event.TripID,
			"driver_id":          event.DriverID,
			"expires_in_seconds": 15,
			"timestamp":          time.Now().UnixMilli(),
		}
		if err := k.centrifugo.Publish(ctx, driverChannel, dispatchOfferPayload); err != nil {
			logger.Warn(ctx, "Failed to publish dispatch offer to driver Centrifugo channel",
				"driver_id", event.DriverID, "error", err)
		} else {
			logger.Info(ctx, "Published dispatch offer to driver Centrifugo channel",
				"channel", driverChannel, "trip_id", event.TripID)
		}
	}
}
