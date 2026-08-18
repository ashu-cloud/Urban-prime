package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cab-booking/payment-service/internal/config"
	"github.com/cab-booking/payment-service/internal/handler"
	"github.com/cab-booking/payment-service/internal/repository"
	"github.com/cab-booking/payment-service/internal/stripeclient"
	"github.com/cab-booking/pkg/logger"
	paymentv1 "github.com/cab-booking/proto/gen/payment/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Initialize Logger
	logger.Init("payment-service")

	// Connect to Database
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Init DB Schema (Simple migration for dev)
	initSchema(ctx, dbPool)

	// Initialize Repository
	repo := repository.NewTransactionRepository(dbPool)

	// Initialize Stripe Client
	stripeClient := stripeclient.NewClient(cfg.StripeSecretKey)

	// Initialize gRPC Handler
	grpcHandler := handler.NewPaymentHandler(repo, stripeClient)

	// Setup gRPC Server
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Graceful Shutdown
	go func() {
		logger.Info(ctx, "Starting Payment Service", "port", cfg.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(ctx, "Shutting down Payment Service")
	grpcServer.GracefulStop()
}

func initSchema(ctx context.Context, pool *pgxpool.Pool) {
	schema := `
	CREATE TABLE IF NOT EXISTS transactions (
		id UUID PRIMARY KEY,
		trip_id UUID NOT NULL,
		rider_id UUID NOT NULL,
		amount_cents BIGINT NOT NULL,
		currency VARCHAR(10) NOT NULL,
		stripe_payment_intent_id VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_transactions_trip_id ON transactions (trip_id);
	`
	if _, err := pool.Exec(ctx, schema); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}
}
