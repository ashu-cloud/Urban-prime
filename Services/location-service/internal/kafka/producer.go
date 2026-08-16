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

// TopicDriverLocation is the Kafka topic that carries all driver GPS update events.
// The Notification Service consumes from this topic to push real-time map updates.
const TopicDriverLocation = "driver.location.v1"

// LocationEvent is the Kafka message payload published for every GPS ping from a driver.
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

// Producer publishes LocationEvent messages to Kafka asynchronously.
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a Kafka producer for the Location Service
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

// PublishLocationUpdate serialises and publishes a driver GPS update event to Kafka.
// Key = driver_id ensures all pings from the same driver go to the same partition
// (maintains ordering of GPS coordinates per driver).
func (p *Producer) PublishLocationUpdate(ctx context.Context, event LocationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal location event: %w", err)
	}

	record := &kgo.Record{
		Topic: TopicDriverLocation,
		Key:   []byte(event.DriverID), // keyed by driver_id for ordered delivery per driver
		Value: data,
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if p.client != nil {
		results := p.client.ProduceSync(ctxTimeout, record)
		if err := results.FirstErr(); err != nil {
			// Non-fatal: location data is already written to Redis Geo. Kafka is for downstream fanout.
			logger.Warn(ctx, "Kafka location event publish failed (Redis already updated)",
				"driver_id", event.DriverID,
				"error", err,
			)
		}
	}

	return nil
}

func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
