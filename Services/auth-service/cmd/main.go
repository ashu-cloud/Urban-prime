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

	"github.com/cab-booking/auth-service/internal/config"
	"github.com/cab-booking/auth-service/internal/handler"
	jwtmgr "github.com/cab-booking/auth-service/internal/jwt"
	"github.com/cab-booking/auth-service/internal/repository"
	"github.com/cab-booking/pkg/logger"
	authv1 "github.com/cab-booking/proto/gen/auth/v1"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	logger.Info(ctx, "Initializing Auth Service...", "http_port", cfg.HTTPPort, "grpc_port", cfg.GRPCPort)

	// 1. Database connection pool
	pool, err := initDatabase(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL", "error", err)
	} else {
		defer pool.Close()
		runMigrations(ctx, cfg.DatabaseDSN)
	}

	// 2. Repositories & Managers
	var repo *repository.UserRepository
	if pool != nil {
		repo = repository.NewUserRepository(pool)
	}

	tokenManager := jwtmgr.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTLMin, cfg.JWTRefreshTTLDays)

	// 3. HTTP REST Handler & Server
	httpHandler := handler.NewHTTPHandler(repo, tokenManager)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", httpHandler.Register)
	mux.HandleFunc("/auth/login", httpHandler.Login)
	mux.HandleFunc("/auth/refresh", httpHandler.Refresh)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		logger.Info(ctx, "Auth Service HTTP server listening", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(ctx, "HTTP server stopped unexpectedly", "error", err)
		}
	}()

	// 4. gRPC Handler & Server
	grpcHandler := handler.NewGRPCHandler(tokenManager)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		logger.Error(ctx, "Failed to listen on gRPC port", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	go func() {
		logger.Info(ctx, "Auth Service gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error(ctx, "gRPC server stopped unexpectedly", "error", err)
		}
	}()

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down Auth Service gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()

	logger.Info(ctx, "Auth Service stopped cleanly")
}

func initDatabase(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}
	config.MaxConns = 15
	config.MinConns = 3

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

	logger.Info(ctx, "Connected to PostgreSQL database successfully (Auth Service)")
	return pool, nil
}

func runMigrations(ctx context.Context, dsn string) {
	logger.Info(ctx, "Running database migrations for auth-service...")
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Warn(ctx, "Failed to initialize SQL migrations", "error", err)
		return
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Warn(ctx, "Migration process returned notice", "error", err)
	} else {
		logger.Info(ctx, "Auth database migrations executed successfully")
	}
}
