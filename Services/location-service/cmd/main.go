package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	locationv1 "github.com/cab-booking/proto/gen/location/v1"
	"github.com/cab-booking/location-service/internal/config"
	"github.com/cab-booking/location-service/internal/geo"
	"github.com/cab-booking/location-service/internal/handler"
	"github.com/cab-booking/location-service/internal/kafka"
	"github.com/cab-booking/pkg/logger"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// main() is the entry point for the Location Service — the GPS firehose ingestor.
//
// This service handles 100,000 concurrent drivers pinging their GPS every 3 seconds
// = ~33,000 location updates per second at peak load.
//
// Design choices for scalability:
//   - NO database writes on the hot path — only Redis Geo in-memory writes (~1ms)
//   - Kafka publish is fire-and-forget (non-blocking, best-effort)
//   - gRPC server with connection pooling handles concurrent pings efficiently
func main() {
	ctx := context.Background()
	cfg := config.Load()

	logger.Info(ctx, "Initializing Location Service (GPS Firehose Ingestor)...", "port", cfg.Port)

	// 1. REDIS SETUP — the in-memory Geo spatial index
	// Every GPS ping from a driver writes here. The Driver Service reads here for dispatch matching.
	redisClient := initRedis(ctx, cfg.RedisAddr)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 2. KAFKA PRODUCER — for downstream real-time WebSocket fanout
	// Notification Service consumes these events and pushes live map updates to rider apps via Centrifugo
	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Warn(ctx, "Kafka producer init with warning (location events will be Redis-only)", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	// 3. GEO CLIENT — wraps Redis Geo commands (GEOADD, GEOPOS)
	geoClient := geo.NewGeoClient(redisClient)

	// 4. gRPC HANDLER — the hot path that handles every GPS ping
	locationHandler := handler.NewLocationHandler(geoClient, producer)

	// 5. START gRPC SERVER ON PORT 50053
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		logger.Error(ctx, "Failed to listen on port", "port", cfg.Port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	locationv1.RegisterLocationServiceServer(grpcServer, locationHandler)
	reflection.Register(grpcServer) // enables grpcurl / Postman introspection

	go func() {
		logger.Info(ctx, "Location Service gRPC server listening for GPS pings", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error(ctx, "gRPC server stopped unexpectedly", "error", err)
		}
	}()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: locationHandler.Routes(),
	}
	go func() {
		logger.Info(ctx, "Location Service HTTP REST adapter listening", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(ctx, "HTTP server stopped unexpectedly", "error", err)
		}
	}()

	// 6. GRACEFUL SHUTDOWN — wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Location Service gracefully...")
	_ = httpServer.Shutdown(context.Background())
	grpcServer.GracefulStop()
	logger.Info(ctx, "Location Service stopped cleanly")
}

// initRedis connects to Redis and validates the connection.
// Returns nil (with a warning) if Redis is unavailable — service degrades gracefully.
func initRedis(ctx context.Context, addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     50, // large pool — this service has HIGH write throughput
	})

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(ctxTimeout).Err(); err != nil {
		logger.Warn(ctx, "Redis unavailable — GPS pings will not be persisted until Redis reconnects", "addr", addr, "error", err)
		return nil
	}

	logger.Info(ctx, "Connected to Redis Geo spatial index (Location Service)", "addr", addr)
	return client
}
