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

	driverv1 "github.com/cab-booking/proto/gen/driver/v1"
	"github.com/cab-booking/driver-service/internal/config"
	"github.com/cab-booking/driver-service/internal/dispatch"
	"github.com/cab-booking/driver-service/internal/geo"
	"github.com/cab-booking/driver-service/internal/handler"
	"github.com/cab-booking/driver-service/internal/kafka"
	"github.com/cab-booking/driver-service/internal/repository"
	"github.com/cab-booking/pkg/logger"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	logger.Info(ctx, "Initializing Driver Service (Matchmaking Engine)...", "port", cfg.Port)

	// 1. DATABASE CONNECTION
	pool, err := initDatabase(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL", "error", err)
	} else {
		defer pool.Close()
		runMigrations(ctx, cfg.DatabaseDSN)
	}

	// 2. REDIS SETUP
	redisClient := initRedis(ctx, cfg.RedisAddr)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 3. REPOSITORIES & SERVICES
	var repo *repository.DriverRepository
	if pool != nil {
		repo = repository.NewDriverRepository(pool)
	}

	geoService := geo.NewGeoService(redisClient)

	// 4. KAFKA PRODUCER
	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Warn(ctx, "Kafka producer initialized with warning", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	// 5. DISPATCH LOOP ENGINE
	dispatchLoop := dispatch.NewDispatchLoop(geoService, repo, producer)

	// 6. KAFKA CONSUMER (Listens for `trip.events.v1 { MATCHING }` to trigger dispatch loop)
	dispatchAdapter := func(c context.Context, tripID string, pickupLat, pickupLng float64, vehicleType string) error {
		_, err := dispatchLoop.FindAndDispatchDriver(c, tripID, pickupLat, pickupLng, vehicleType)
		return err
	}

	consumer, err := kafka.NewConsumer(cfg.KafkaBrokers, "driver-service-group", dispatchAdapter)
	if err != nil {
		logger.Warn(ctx, "Kafka consumer init warning", "error", err)
	}
	if consumer != nil {
		go consumer.Start(ctx)
		logger.Info(ctx, "Driver Service Kafka consumer loop LIVE — ready to match trips")
	}

	// 7. gRPC SERVICE HANDLER
	driverHandler := handler.NewDriverHandler(dispatchLoop, geoService, repo)

	// 8. START gRPC SERVER ON PORT 50052
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		logger.Error(ctx, "Failed to listen on port", "port", cfg.Port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	driverv1.RegisterDriverServiceServer(grpcServer, driverHandler)
	reflection.Register(grpcServer)

	go func() {
		logger.Info(ctx, "Driver Service gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error(ctx, "gRPC server stopped unexpectedly", "error", err)
		}
	}()

	// 9. GRACEFUL SHUTDOWN
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Driver Service gracefully...")
	cancel()
	grpcServer.GracefulStop()
	logger.Info(ctx, "Driver Service stopped cleanly")
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

func initRedis(ctx context.Context, addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(ctxTimeout).Err(); err != nil {
		logger.Warn(ctx, "Redis ping failed (Driver Service running with mock in-memory fallback)", "addr", addr, "error", err)
		return nil
	}

	logger.Info(ctx, "Connected to Redis Geo spatial index successfully", "addr", addr)
	return client
}

func runMigrations(ctx context.Context, dsn string) {
	logger.Info(ctx, "Running database migrations for driver-service...")
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Warn(ctx, "Failed to initialize SQL migrations", "error", err)
		return
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Warn(ctx, "Migration process returned notice", "error", err)
	} else {
		logger.Info(ctx, "Driver database migrations executed successfully")
	}
}
