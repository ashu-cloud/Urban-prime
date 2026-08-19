.PHONY: help dev up down backend frontend restart status logs proto tidy build clean

help:
	@echo ""
	@echo "=========================================================="
	@echo "  🚕 Urban Prime Mobility OS - Quick Make Commands"
	@echo "=========================================================="
	@echo ""
	@echo "  make dev        - 🚀 1-Command Dev: Starts Docker infra + all 6 Go services"
	@echo "  make up         - Start all Docker containers (DB, Redis, Kafka, APISIX, Centrifugo, Frontend)"
	@echo "  make down       - Stop all Docker containers"
	@echo "  make backend    - Run all 6 Go microservices concurrently with colored logs"
	@echo "  make frontend   - Run Next.js frontend in local dev mode (npm run dev)"
	@echo "  make restart    - Restart all Docker containers"
	@echo "  make status     - View status & health of all containers"
	@echo "  make logs       - Tail live container logs"
	@echo "  make proto      - Recompile Protobuf definitions"
	@echo "  make tidy       - Run go mod tidy across all modules"
	@echo "  make build      - Build all Go service binaries"
	@echo ""

# 1-Command Full Dev Environment
dev: up
	@echo ""
	@echo "⚡ Infrastructure is ready. Launching all 6 Go Microservices..."
	@echo ""
	go run ./devserver/main.go

# Start Docker Infrastructure & Frontend
up:
	docker-compose -f deploy/docker-compose.yml up -d

# Stop Docker Infrastructure
down:
	docker-compose -f deploy/docker-compose.yml down

# Restart Docker Infrastructure
restart: down up

# Run all 6 Go microservices in 1 terminal with color-coded live logs
backend:
	go run ./devserver/main.go

# Run Next.js Frontend with Turbopack in local dev mode
frontend:
	cd frontend && npm run dev

# Show Docker Container Status
status:
	docker-compose -f deploy/docker-compose.yml ps

# Tail Live Docker Container Logs
logs:
	docker-compose -f deploy/docker-compose.yml logs -f

# Generate Protobuf Code
proto:
	protoc --go_out=proto/gen --go_opt=paths=source_relative \
	       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
	       -I=proto proto/auth/v1/auth.proto proto/trip/v1/trip.proto \
	       proto/driver/v1/driver.proto proto/location/v1/location.proto \
	       proto/payment/v1/payment.proto

# Tidy all Go modules
tidy:
	cd pkg && go mod tidy
	cd Services/auth-service && go mod tidy
	cd Services/trip-service && go mod tidy
	cd Services/driver-service && go mod tidy
	cd Services/location-service && go mod tidy
	cd Services/payment-service && go mod tidy
	cd Services/notification-service && go mod tidy
	cd devserver && go mod tidy

# Build all Go service binaries into /bin
build: tidy
	cd Services/auth-service && go build -o bin/auth-service ./cmd
	cd Services/trip-service && go build -o bin/trip-service ./cmd
	cd Services/driver-service && go build -o bin/driver-service ./cmd
	cd Services/location-service && go build -o bin/location-service ./cmd
	cd Services/payment-service && go build -o bin/payment-service ./cmd
	cd Services/notification-service && go build -o bin/notification-service ./cmd
