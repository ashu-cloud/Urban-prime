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

// Consumer listens to `driver.match.v1` and advances the Trip Saga state machine
type Consumer struct {
	client                 *kgo.Client
	assignDriverFunc       AssignDriverFunc
	compensateNoDriverFunc CompensateNoDriverFunc
}

func NewConsumer(brokers, groupID string, assignDriverFunc AssignDriverFunc, compensateNoDriverFunc CompensateNoDriverFunc) (*Consumer, error) {
	brokerList := strings.Split(brokers, ",")
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(TopicMatchEvents),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trip service kafka consumer: %w", err)
	}

	return &Consumer{
		client:                 client,
		assignDriverFunc:       assignDriverFunc,
		compensateNoDriverFunc: compensateNoDriverFunc,
	}, nil
}

// Start begins continuous event listening loop in background
func (c *Consumer) Start(ctx context.Context) {
	logger.Info(ctx, "Trip Service Kafka consumer started — listening for driver match events", "topic", TopicMatchEvents)

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Trip Service Kafka consumer stopping...")
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
			var event matchEventMessage
			if err := json.Unmarshal(record.Value, &event); err != nil {
				logger.Warn(ctx, "Failed to unmarshal match event in Trip Service", "error", err)
				return
			}

			switch event.EventType {
			case "ACCEPTED":
				logger.Info(ctx, "Saga Step 3 Completed: Driver ACCEPTED trip", "trip_id", event.TripID, "driver_id", event.DriverID)
				if c.assignDriverFunc != nil {
					if err := c.assignDriverFunc(ctx, event.TripID, event.DriverID); err != nil {
						logger.Error(ctx, "Failed to assign driver to trip in Saga Orchestrator", "trip_id", event.TripID, "error", err)
					}
				}

			case "EXHAUSTED":
				logger.Warn(ctx, "Saga Compensation Triggered: All candidate drivers exhausted", "trip_id", event.TripID)
				if c.compensateNoDriverFunc != nil {
					c.compensateNoDriverFunc(ctx, event.TripID)
				}
			}
		})
	}
}
