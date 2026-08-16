package main

// IMPORTS SECTION:
// In Go, external libraries and internal packages are imported here.
// Go automatically groups standard library imports ("context", "net", etc.)
// from third-party and local workspace packages.
import (
	"context" // Core Go package for passing deadlines, cancellations, and request context across APIs
	"errors"  // Helper functions to create and compare errors
	"fmt"     // Format package for string formatting and printing (like String.format in Java/C#)
	"net"     // Network I/O primitive package used here to create a TCP network socket listener
	"os"      // Interacting with the Operating System (env vars, exit codes, process management)
	"os/signal" // Captures OS hardware signals (like Ctrl+C or kill signals)
	"syscall"   // System call constants (SIGINT, SIGTERM)
	"time"      // Time units, durations, timers, and timestamps

	// Local workspace modules created for our cab booking system:
	"github.com/cab-booking/pkg/logger"               // Our custom JSON logger
	tripv1 "github.com/cab-booking/proto/gen/trip/v1" // Generated gRPC interfaces & protobuf types
	"github.com/cab-booking/trip-service/internal/config"     // Environment variable loader
	"github.com/cab-booking/trip-service/internal/handler"    // gRPC request handlers
	"github.com/cab-booking/trip-service/internal/kafka"      // Kafka message producer
	"github.com/cab-booking/trip-service/internal/osrm"       // OSRM map routing HTTP client
	"github.com/cab-booking/trip-service/internal/pricing"    // Uber-model fare calculation engine
	"github.com/cab-booking/trip-service/internal/repository" // Raw SQL PostgreSQL query handler
	"github.com/cab-booking/trip-service/internal/saga"       // Saga transaction orchestrator

	// Third-party open-source libraries:
	"github.com/golang-migrate/migrate/v4" // Automatic SQL migration runner
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Blank import (`_`) registers Postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Blank import (`_`) registers file system driver
	"github.com/jackc/pgx/v5/pgxpool"                           // High-performance PostgreSQL connection pool
	"google.golang.org/grpc"                                    // Google gRPC server engine
	"google.golang.org/grpc/reflection"                         // Enables Postman/grpcurl dynamic API inspection
)

// main() is the entry point of our Go microservice binary.
func main() {
	// 1. Create a background context. Context carries deadlines and signals across function calls.
	ctx := context.Background()

	// 2. Load configuration parameters from environment variables (or defaults from .env.example)
	cfg := config.Load()

	logger.Info(ctx, "Initializing Trip Service...", "port", cfg.Port)

	// 3. DATABASE SETUP:
	// We create a connection pool to PostgreSQL. A connection pool reuses database connections
	// instead of opening a expensive new database connection for every incoming user request.
	pool, err := initDatabase(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL (Service running without active DB connection)", "error", err)
	} else {
		// 'defer' ensures pool.Close() is automatically called right before main() exits.
		defer pool.Close()

		// Run SQL migrations automatically on startup to ensure tables (`trips`) exist.
		runMigrations(ctx, cfg.DatabaseDSN)
	}

	// 4. DEPENDENCY INJECTION (DI):
	// In Go, dependencies are explicitly passed into structs via constructors (New... functions).

	// Initialize database repository if database connection pool is active
	var repo *repository.TripRepository
	if pool != nil {
		repo = repository.NewTripRepository(pool)
	}

	// Initialize OSRM routing client (calls OSRM API for driving distances)
	osrmClient := osrm.NewClient(cfg.OSRMHost)

	// Initialize Uber fare pricing calculator (base + km + time formula)
	calc := pricing.NewCalculator(cfg)

	// Initialize Kafka event producer to broadcast trip events asynchronously
	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		logger.Warn(ctx, "Kafka producer initialized with warning", "error", err)
	}
	if producer != nil {
		defer producer.Close() // Automatically close Kafka connection on shutdown
	}

	// 5. SAGA ORCHESTRATOR & gRPC HANDLER:
	// Assemble the Saga Orchestrator with its required dependencies (Repo, OSRM, Calculator, Kafka)
	orchestrator := saga.NewOrchestrator(repo, osrmClient, calc, producer)

	// Create the gRPC Handler struct that receives gRPC requests from riders/gateway
	tripHandler := handler.NewTripHandler(orchestrator, repo)

	// 6. START gRPC SERVER:
	// Open a TCP network listener on port 50051 (e.g., "0.0.0.0:50051")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		logger.Error(ctx, "Failed to listen on port", "port", cfg.Port, "error", err)
		os.Exit(1)
	}

	// Initialize gRPC server instance
	grpcServer := grpc.NewServer()

	// Register our implementation of TripServiceServer with the gRPC server
	tripv1.RegisterTripServiceServer(grpcServer, tripHandler)

	// Enable gRPC Reflection (allows tools like Postman to inspect endpoints dynamically without importing .proto files)
	reflection.Register(grpcServer)

	// 7. CONCURRENCY & GRACEFUL SHUTDOWN:
	// In Go, the 'go' keyword starts a lightweight thread called a "goroutine".
	// We run the gRPC server in a separate background goroutine so it doesn't block the main thread.
	go func() {
		logger.Info(ctx, "Trip Service gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error(ctx, "gRPC server stopped unexpectedly", "error", err)
		}
	}()

	// 8. WAIT FOR SHUTDOWN SIGNAL:
	// Go channels (chan) allow goroutines to communicate.
	// We create a channel that receives OS signals (Ctrl+C or Docker stop).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// This line BLOCKS execution until an OS termination signal is received on the channel.
	<-quit

	logger.Info(ctx, "Shutting down Trip Service gracefully...")

	// GracefulStop waits for active gRPC requests to complete before closing listeners.
	grpcServer.GracefulStop()

	logger.Info(ctx, "Trip Service stopped cleanly")
}

// initDatabase creates and tests a high-performance pgx pool connection to PostgreSQL
func initDatabase(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}

	// Connection Pool limits to prevent overwhelming database under peak loads
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 15 * time.Minute

	// Create a short 5-second timeout context for establishing initial connection
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctxTimeout, config)
	if err != nil {
		return nil, err
	}

	// Ping database to verify connection is alive
	if err := pool.Ping(ctxTimeout); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info(ctx, "Connected to PostgreSQL database successfully")
	return pool, nil
}

// runMigrations automatically executes database migrations (.up.sql files) on startup
func runMigrations(ctx context.Context, dsn string) {
	logger.Info(ctx, "Running database migrations...")

	// golang-migrate reads SQL files from the local filesystem directory "file://migrations"
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Warn(ctx, "Failed to initialize SQL migrations", "error", err)
		return
	}
	defer m.Close()

	// Apply all pending '.up.sql' migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Warn(ctx, "Migration process returned notice", "error", err)
	} else {
		logger.Info(ctx, "Database migrations executed successfully")
	}
}
