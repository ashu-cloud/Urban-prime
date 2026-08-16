package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cab-booking/pkg/logger"
	"github.com/twmb/franz-go/pkg/kgo"
)

// MatchEventPayload represents match dispatch lifecycle events emitted to Kafka `driver.match.v1`
type MatchEventPayload struct {
	EventType string    `json:"event_type"` // "OFFERED", "ACCEPTED", "DECLINED", "EXHAUSTED"
	TripID    string    `json:"trip_id"`
	DriverID  string    `json:"driver_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Attempts  int       `json:"attempts,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Producer struct {
	client *kgo.Client
}

func NewProducer(brokers string) (*Producer, error) {
	brokerList := strings.Split(brokers, ",")
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Producer{client: client}, nil
}

// PublishMatchEvent produces match lifecycle events synchronously to Kafka topic `driver.match.v1`
func (p *Producer) PublishMatchEvent(ctx context.Context, topic string, payload MatchEventPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal match event payload: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(payload.TripID),
		Value: data,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if p.client != nil {
		results := p.client.ProduceSync(ctxTimeout, record)
		if err := results.FirstErr(); err != nil {
			logger.Warn(ctx, "Kafka broker unavailable, logging match event locally", "topic", topic, "trip_id", payload.TripID, "event", payload.EventType, "error", err)
		} else {
			logger.Info(ctx, "Published Kafka match event successfully", "topic", topic, "trip_id", payload.TripID, "event", payload.EventType)
		}
	}

	return nil
}

func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
