package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tripv1 "github.com/cab-booking/proto/gen/trip/v1"
	"github.com/cab-booking/pkg/logger"
	"github.com/cab-booking/trip-service/internal/config"
	"github.com/cab-booking/trip-service/internal/handler"
	"github.com/cab-booking/trip-service/internal/kafka"
	"github.com/cab-booking/trip-service/internal/osrm"
	"github.com/cab-booking/trip-service/internal/pricing"
	"github.com/cab-booking/trip-service/internal/repository"
	"github.com/cab-booking/trip-service/internal/saga"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	logger.Info(ctx, "Initializing Trip Service...", "port", cfg.Port, "osrm_host", cfg.OSRMHost)

	// 1. DATABASE CONNECTION
	pool, err := initDatabase(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL", "error", err)
	} else {
		defer pool.Close()
		runMigrations(ctx, cfg.DatabaseDSN)
	}

	// 2. DEPENDENCY INJECTION
	var repo *repository.TripRepository
	if pool != nil {
		repo = repository.NewTripRepository(pool)
	}

	osrmClient := osrm.NewClient(cfg.OSRMHost)
	calculator := pricing.NewCalculator(cfg)

	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Warn(ctx, "Kafka producer initialized with warning", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	// 3. SAGA ORCHESTRATOR
	orchestrator := saga.NewOrchestrator(repo, osrmClient, calculator, producer)

	// 4. KAFKA CONSUMER (Listens for `driver.match.v1 { ACCEPTED / EXHAUSTED }` to update trip status)
	consumer, err := kafka.NewConsumer(
		cfg.KafkaBrokers,
		"trip-service-group",
		orchestrator.AssignDriverToTrip,
		orchestrator.CompensateNoDriverAvailable,
	)
	if err != nil {
		logger.Warn(ctx, "Trip Service Kafka consumer init warning", "error", err)
	}
	if consumer != nil {
		go consumer.Start(ctx)
		logger.Info(ctx, "Trip Service Kafka consumer loop LIVE — listening for driver match outcomes")
	}

	// 5. gRPC HANDLER
	tripHandler := handler.NewTripHandler(orchestrator, repo)

	// 6. START gRPC SERVER ON PORT 50051
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		logger.Error(ctx, "Failed to listen on port", "port", cfg.Port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	tripv1.RegisterTripServiceServer(grpcServer, tripHandler)
	reflection.Register(grpcServer)

	go func() {
		logger.Info(ctx, "Trip Service gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error(ctx, "gRPC server stopped unexpectedly", "error", err)
		}
	}()

	// 7. GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Trip Service gracefully...")
	cancel()
	grpcServer.GracefulStop()
	logger.Info(ctx, "Trip Service stopped cleanly")
}

func initDatabase(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}
	config.MaxConns = 25
	config.MinConns = 5

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctxTimeout, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctxTimeout); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info(ctx, "Connected to PostgreSQL database successfully")
	return pool, nil
}

func runMigrations(ctx context.Context, dsn string) {
	logger.Info(ctx, "Running database migrations for trip-service...")
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Warn(ctx, "Failed to initialize SQL migrations", "error", err)
		return
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Warn(ctx, "Migration process returned notice", "error", err)
	} else {
		logger.Info(ctx, "Database migrations executed successfully")
	}
}
