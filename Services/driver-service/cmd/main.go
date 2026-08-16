package main

// IMPORTS SECTION:
// Go standard library, internal workspace packages, and third-party drivers
import (
	"context" // Manages request context, cancellation signals, and deadlines
	"errors"  // Helper for error checking
	"fmt"     // String formatting functions
	"net"     // Network TCP listener primitives
	"os"      // Environment variables and process management
	"os/signal" // OS interrupt signal handler
	"syscall"   // System call constants (SIGINT, SIGTERM)
	"time"      // Time units and timeouts

	// Local workspace modules:
	driverv1 "github.com/cab-booking/proto/gen/driver/v1" // Generated gRPC interfaces & protobuf types
	"github.com/cab-booking/driver-service/internal/config"   // Environment loader
	"github.com/cab-booking/driver-service/internal/dispatch" // Matchmaking dispatch loop engine
	"github.com/cab-booking/driver-service/internal/geo"      // Redis Geo spatial indexing service
	"github.com/cab-booking/driver-service/internal/handler"  // gRPC service request handler
	"github.com/cab-booking/driver-service/internal/kafka"    // Kafka match event producer
	"github.com/cab-booking/driver-service/internal/repository" // Raw SQL PostgreSQL driver repository
	"github.com/cab-booking/pkg/logger"                       // Custom JSON structured logger

	// Third-party open-source libraries:
	"github.com/golang-migrate/migrate/v4" // SQL database migration runner
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Blank import (`_`) registers Postgres migration driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Blank import (`_`) registers local file driver
	"github.com/jackc/pgx/v5/pgxpool"                           // High-performance PostgreSQL connection pool
	"github.com/redis/go-redis/v9"                             // Official Redis client supporting Geo commands
	"google.golang.org/grpc"                                    // Google gRPC framework
	"google.golang.org/grpc/reflection"                         // EnablesPostman/grpcurl dynamic API inspection
)

// main() is the entry point for the Driver Service microservice binary.
func main() {
	ctx := context.Background()
	cfg := config.Load()

	logger.Info(ctx, "Initializing Driver Service (Matchmaking Engine)...", "port", cfg.Port)

	// 1. DATABASE CONNECTION (PostgreSQL Pool)
	pool, err := initDatabase(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL", "error", err)
	} else {
		defer pool.Close()
		runMigrations(ctx, cfg.DatabaseDSN)
	}

	// 2. IN-MEMORY REDIS SPATIAL INDEX SETUP
	redisClient := initRedis(ctx, cfg.RedisAddr)
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 3. REPOSITORIES & SERVICES INJECTION
	var repo *repository.DriverRepository
	if pool != nil {
		repo = repository.NewDriverRepository(pool)
	}

	geoService := geo.NewGeoService(redisClient)

	// 4. KAFKA EVENT PRODUCER
	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Warn(ctx, "Kafka producer initialized with warning", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	// 5. DISPATCH LOOP ENGINE
	// The Dispatch Loop is the core matchmaking engine that searches Redis Geo for nearest drivers,
	// offers rides sequentially, enforces timeouts, and falls back to next nearest candidates.
	dispatchLoop := dispatch.NewDispatchLoop(geoService, repo, producer)

	// 6. gRPC SERVICE HANDLER
	driverHandler := handler.NewDriverHandler(dispatchLoop, geoService, repo)

	// 7. START gRPC SERVER ON PORT 50052
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

	// 8. GRACEFUL SHUTDOWN HANDLING
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Driver Service gracefully...")
	grpcServer.GracefulStop()
	logger.Info(ctx, "Driver Service stopped cleanly")
}

// initDatabase creates and validates PostgreSQL pgx connection pool
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

// initRedis initializes connections to Redis in-memory datastore
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

// runMigrations executes SQL migrations (.up.sql files) on startup
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
