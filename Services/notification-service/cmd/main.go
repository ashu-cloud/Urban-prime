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
//  2. Location Service writes GPS → Redis Geo + Redis Stream `driver.location.v1`
//  3. THIS service (Notification Service) consumes from Redis Streams
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
		"redis_addr", cfg.RedisAddr,
	)

	// 1. CENTRIFUGO HTTP CLIENT — for pushing real-time WebSocket messages
	centrifugoClient := centrifugo.NewClient(cfg.CentrifugoURL, cfg.CentrifugoAPIKey)

	// 2. REDIS STREAM CONSUMER — reads location, trip, and match events from Redis Streams
	streamConsumer, err := consumer.NewKafkaConsumer(cfg.RedisAddr, cfg.ConsumerGroupID, centrifugoClient)
	if err != nil {
		logger.Warn(ctx, "Redis stream consumer init warning — will retry connections automatically", "error", err)
	}

	// 3. START THE EVENT CONSUMPTION LOOP (runs as a blocking goroutine)
	// This is the core loop: Redis Stream events → Centrifugo WebSocket push
	if streamConsumer != nil {
		go streamConsumer.Start(ctx)
		logger.Info(ctx, "Redis stream consumer started — real-time event pipeline is LIVE 🚀")
	}

	// 4. GRACEFUL SHUTDOWN — wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Notification Service gracefully...")
	cancel() // signal the stream consumer loop to stop
	logger.Info(ctx, "Notification Service stopped cleanly")
}
