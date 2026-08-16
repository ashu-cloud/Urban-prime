package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cab-booking/pkg/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
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

// Consumer listens to `trip.events.v1` and triggers the Driver Service dispatch loop
type Consumer struct {
	client       *kgo.Client
	dispatchFunc DispatchFunc
}

func NewConsumer(brokers, groupID string, dispatchFunc DispatchFunc) (*Consumer, error) {
	brokerList := strings.Split(brokers, ",")
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(TopicTripEvents),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver service kafka consumer: %w", err)
	}

	return &Consumer{
		client:       client,
		dispatchFunc: dispatchFunc,
	}, nil
}

// Start begins continuous event listening loop in background
func (c *Consumer) Start(ctx context.Context) {
	logger.Info(ctx, "Driver Service Kafka consumer started — listening for trip MATCHING events", "topic", TopicTripEvents)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Driver Service Kafka consumer stopping...")
			if c.client != nil {
				c.client.Close()
			}
			return
		default:
		}

		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return
		}

		fetches.EachRecord(func(record *kgo.Record) {
			var event tripEventMessage
			if err := json.Unmarshal(record.Value, &event); err != nil {
				logger.Warn(ctx, "Failed to unmarshal trip event in Driver Service", "error", err)
				return
			}

			// WHEN A TRIP ENTERS 'MATCHING' STATUS → TRIGGER DISPATCH LOOP!
			if event.Status == "MATCHING" && c.dispatchFunc != nil {
				logger.Info(ctx, "Kafka Event Received: Trip in MATCHING state — triggering Dispatch Loop", "trip_id", event.TripID)
				go func(e tripEventMessage) {
					dispatchCtx := context.Background()
					err := c.dispatchFunc(dispatchCtx, e.TripID, e.PickupLat, e.PickupLng, e.VehicleType)
					if err != nil {
						logger.Error(dispatchCtx, "Dispatch loop execution returned error", "trip_id", e.TripID, "error", err)
					}
				}(event)
			}
		})
	}
}
