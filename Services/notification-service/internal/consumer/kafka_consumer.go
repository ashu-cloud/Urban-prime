package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cab-booking/notification-service/internal/centrifugo"
	"github.com/cab-booking/pkg/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Topics consumed by the Notification Service
const (
	TopicDriverLocation = "driver.location.v1" // GPS pings from active drivers
	TopicTripEvents     = "trip.events.v1"      // Trip lifecycle events (created, matched, completed)
	TopicMatchEvents    = "driver.match.v1"     // Driver dispatch events (offered, accepted, declined)
)

// locationEvent mirrors the payload published by the Location Service to Kafka
type locationEvent struct {
	DriverID  string    `json:"driver_id"`
	TripID    string    `json:"trip_id,omitempty"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	SpeedKmh  float32   `json:"speed_kmh,omitempty"`
	Bearing   float32   `json:"bearing,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// tripEvent mirrors the payload published by the Trip Service to Kafka
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

// matchEvent mirrors the payload published by the Driver Service to Kafka
type matchEvent struct {
	EventType string    `json:"event_type"` // OFFERED, ACCEPTED, DECLINED, EXHAUSTED
	TripID    string    `json:"trip_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// KafkaConsumer is the core of the Notification Service.
// It reads events from Kafka and translates them into Centrifugo WebSocket broadcasts.
//
// HOW IT SCALES:
// Kafka consumer groups allow multiple instances of the Notification Service to run in parallel.
// Kafka automatically distributes partitions across instances — no duplicate message processing!
// This means you can scale the Notification Service horizontally to handle more throughput.
type KafkaConsumer struct {
	client      *kgo.Client           // Kafka consumer client
	centrifugo  *centrifugo.Client    // Centrifugo HTTP publish client
}

// NewKafkaConsumer creates a new Kafka consumer group member
func NewKafkaConsumer(brokers, groupID string, centrifugoClient *centrifugo.Client) (*KafkaConsumer, error) {
	brokerList := strings.Split(brokers, ",")

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
		// Consumer group ID — Kafka distributes partitions across all members of the same group
		kgo.ConsumerGroup(groupID),
		// Subscribe to all three event topics
		kgo.ConsumeTopics(TopicDriverLocation, TopicTripEvents, TopicMatchEvents),
		// Start consuming from the latest offset (skip historical messages on first start)
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &KafkaConsumer{
		client:     client,
		centrifugo: centrifugoClient,
	}, nil
}

// Start begins the continuous event consumption loop.
// This runs as a goroutine in main.go and processes events indefinitely until ctx is cancelled.
//
// The loop:
//   1. Poll Kafka for a batch of records (blocking with ~100ms timeout)
//   2. For each record, dispatch to the appropriate handler by topic
//   3. Commit offsets back to Kafka (marks messages as processed)
func (k *KafkaConsumer) Start(ctx context.Context) {
	logger.Info(ctx, "Kafka consumer started — listening for real-time events",
		"topics", []string{TopicDriverLocation, TopicTripEvents, TopicMatchEvents},
	)

	for {
		// Check if context was cancelled (service shutdown signal)
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Kafka consumer stopping — context cancelled")
			k.client.Close()
			return
		default:
		}

		// PollFetches blocks until records arrive or timeout (~100ms idle)
		fetches := k.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		// Log any Kafka fetch errors (network issues, partition rebalancing, etc.)
		fetches.EachError(func(t string, p int32, err error) {
			logger.Warn(ctx, "Kafka fetch error", "topic", t, "partition", p, "error", err)
		})

		// Process each record by routing to the correct handler based on the topic
		fetches.EachRecord(func(record *kgo.Record) {
			switch record.Topic {
			case TopicDriverLocation:
				k.handleLocationEvent(ctx, record.Value)
			case TopicTripEvents:
				k.handleTripEvent(ctx, record.Value)
			case TopicMatchEvents:
				k.handleMatchEvent(ctx, record.Value)
			}
		})
	}
}

// handleLocationEvent translates a Kafka location update into a Centrifugo WebSocket push.
// This is the critical path for the "driver moving on map" live-tracking feature!
func (k *KafkaConsumer) handleLocationEvent(ctx context.Context, data []byte) {
	var event locationEvent
	if err := json.Unmarshal(data, &event); err != nil {
		logger.Warn(ctx, "Failed to parse location event", "error", err)
		return
	}

	// Only push live tracking if the driver is on an active trip
	// (rider needs to see driver approaching on the map)
	if event.TripID != "" {
		k.centrifugo.PublishDriverLocation(ctx, event.TripID, event.DriverID, event.Latitude, event.Longitude, event.Bearing)
	}
}

// handleTripEvent translates Kafka trip lifecycle events into Centrifugo WebSocket pushes.
// Rider app receives: "Driver matched!", "Trip started!", "Trip completed!" notifications.
func (k *KafkaConsumer) handleTripEvent(ctx context.Context, data []byte) {
	var event tripEvent
	if err := json.Unmarshal(data, &event); err != nil {
		logger.Warn(ctx, "Failed to parse trip event", "error", err)
		return
	}

	payload := map[string]interface{}{
		"trip_id":   event.TripID,
		"rider_id":  event.RiderID,
		"driver_id": event.DriverID,
		"status":    event.Status,
	}

	// Map trip status to a human-readable event type for the frontend
	eventType := fmt.Sprintf("TRIP_%s", event.Status)
	k.centrifugo.PublishTripEvent(ctx, event.TripID, eventType, payload)

	logger.Info(ctx, "Pushed trip event to Centrifugo channel",
		"trip_id", event.TripID,
		"status", event.Status,
	)
}

// handleMatchEvent handles driver dispatch events (ACCEPTED, DECLINED, EXHAUSTED).
// Rider app receives: "Driver is on the way!" or "No drivers available, please try again."
func (k *KafkaConsumer) handleMatchEvent(ctx context.Context, data []byte) {
	var event matchEvent
	if err := json.Unmarshal(data, &event); err != nil {
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
}
