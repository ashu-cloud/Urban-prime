.PHONY: help dev up up-full down backend frontend restart status logs proto tidy build clean test test-load routes

help:
	@echo ""
	@echo "=========================================================="
	@echo "  🚕 Urban Prime Mobility OS - Quick Make Commands"
	@echo "=========================================================="
	@echo ""
	@echo "  make dev        - 🚀 1-Command Dev: Starts Docker infra + all 6 Go services"
	@echo "  make up         - Start Docker infra only (DB, Redis, Kafka, APISIX, Centrifugo)"
	@echo "  make up-full    - Start infra + backend + frontend containers"
	@echo "  make routes     - Register APISIX routes for /auth and /api/v1"
	@echo "  make down       - Stop all Docker containers"
	@echo "  make backend    - Run all 6 Go microservices concurrently with colored logs"
	@echo "  make frontend   - Run Next.js frontend in local dev mode (npm run dev)"
	@echo "  make restart    - Restart all Docker containers"
	@echo "  make status     - View status & health of all containers"
	@echo "  make logs       - Tail live container logs"
	@echo "  make test       - Run unit, security, concurrency, health, and live probes"
	@echo "  make test-load  - Run k6 load/security/concurrency tests (requires k6)"
	@echo "  make proto      - Recompile Protobuf definitions"
	@echo "  make tidy       - Run go mod tidy across all modules"
	@echo "  make build      - Build all Go service binaries"
	@echo ""

# 1-Command Full Dev Environment (infra in Docker, Go services on the host)
dev: up routes
	@echo ""
	@echo "⚡ Infrastructure is ready. Launching all 6 Go Microservices..."
	@echo ""
	go run ./devserver/main.go

# Start Docker Infrastructure only (so host `make backend` can bind 8080/50051+)
up:
	docker compose -f deploy/docker-compose.yml up -d

# Start infrastructure plus containerized backend + frontend
up-full:
	docker compose -f deploy/docker-compose.yml --profile full up -d --build

# Stop Docker Infrastructure
down:
	docker compose -f deploy/docker-compose.yml --profile full down

# Restart Docker Infrastructure
restart: down up

# Register APISIX routes that match the frontend and k6 scripts
routes:
	powershell -ExecutionPolicy Bypass -File deploy/apisix/setup_routes.ps1

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
	cd tests/live && go mod tidy
	cd proto && go mod tidy
	cd devserver && go mod tidy

# Build all Go service binaries into /bin
build: tidy
	cd Services/auth-service && go build -o bin/auth-service ./cmd
	cd Services/trip-service && go build -o bin/trip-service ./cmd
	cd Services/driver-service && go build -o bin/driver-service ./cmd
	cd Services/location-service && go build -o bin/location-service ./cmd
	cd Services/payment-service && go build -o bin/payment-service ./cmd
	cd Services/notification-service && go build -o bin/notification-service ./cmd

test:
	cd pkg && go test ./... -count=1
	cd Services/auth-service && go test ./... -count=1
	cd Services/trip-service && go test ./... -count=1
	cd Services/driver-service && go test ./... -count=1
	cd Services/location-service && go test ./... -count=1
	cd Services/payment-service && go test ./... -count=1
	cd Services/notification-service && go test ./... -count=1
	cd tests/live && go test ./... -count=1
	cd frontend && npm test

test-load:
	k6 run scripts/load_test_core_flow.js
	k6 run scripts/load_test_security.js
	k6 run scripts/load_test_concurrency.js
