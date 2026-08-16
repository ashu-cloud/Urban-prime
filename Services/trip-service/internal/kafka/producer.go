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

type TripEventPayload struct {
	TripID          string    `json:"trip_id"`
	RiderID         string    `json:"rider_id"`
	Status          string    `json:"status"`
	PickupLat       float64   `json:"pickup_lat"`
	PickupLng       float64   `json:"pickup_lng"`
	DropoffLat      float64   `json:"dropoff_lat"`
	DropoffLng      float64   `json:"dropoff_lng"`
	DistanceKm      float64   `json:"distance_km"`
	EstimatedFare   int64     `json:"estimated_fare_cents"`
	Currency        string    `json:"currency"`
	Timestamp       time.Time `json:"timestamp"`
}

func (p *Producer) PublishTripEvent(ctx context.Context, topic string, payload TripEventPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal trip event payload: %w", err)
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
			logger.Warn(ctx, "Kafka broker unavailable, logging event locally", "topic", topic, "trip_id", payload.TripID, "error", err)
		} else {
			logger.Info(ctx, "Published Kafka event successfully", "topic", topic, "trip_id", payload.TripID, "status", payload.Status)
		}
	}

	return nil
}

func (p *Producer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
