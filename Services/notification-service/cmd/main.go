package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cab-booking/notification-service/internal/centrifugo"
	"github.com/cab-booking/notification-service/internal/config"
	"github.com/cab-booking/notification-service/internal/consumer"
	"github.com/cab-booking/pkg/logger"
)

// main() is the entry point for the Notification Service.
//
// THE BIG PICTURE — HOW REAL-TIME TRACKING WORKS:
//
//  1. Driver's mobile app sends GPS pings to Location Service (gRPC every 3s)
//  2. Location Service writes GPS → Redis Geo + Kafka topic `driver.location.v1`
//  3. THIS service (Notification Service) consumes from Kafka
//  4. It calls Centrifugo HTTP API: POST /api/publish channel="tracking#<trip_id>"
//  5. Centrifugo pushes the message instantly to ALL WebSocket clients on that channel
//  6. Rider's browser map marker updates → 🚖 driver appears moving on the map!
//
// The Notification Service does NOT manage WebSocket connections.
// Centrifugo handles 100k+ concurrent WebSocket client connections independently.
// We just send HTTP POST requests to Centrifugo's simple publish API.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	logger.Info(ctx, "Initializing Notification Service (Centrifugo WebSocket Gateway)...",
		"centrifugo_url", cfg.CentrifugoURL,
		"kafka_brokers", cfg.KafkaBrokers,
	)

	// 1. CENTRIFUGO HTTP CLIENT — for pushing real-time WebSocket messages
	centrifugoClient := centrifugo.NewClient(cfg.CentrifugoURL, cfg.CentrifugoAPIKey)

	// 2. KAFKA CONSUMER — reads location, trip, and match events from Kafka topics
	kafkaConsumer, err := consumer.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaGroupID, centrifugoClient)
	if err != nil {
		logger.Warn(ctx, "Kafka consumer init warning — will retry connections automatically", "error", err)
	}

	// 3. START THE EVENT CONSUMPTION LOOP (runs as a blocking goroutine)
	// This is the core loop: Kafka events → Centrifugo WebSocket push
	if kafkaConsumer != nil {
		go kafkaConsumer.Start(ctx)
		logger.Info(ctx, "Kafka consumer started — real-time event pipeline is LIVE 🚀")
	}

	// 4. GRACEFUL SHUTDOWN — wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Notification Service gracefully...")
	cancel() // signal the Kafka consumer loop to stop
	logger.Info(ctx, "Notification Service stopped cleanly")
}
