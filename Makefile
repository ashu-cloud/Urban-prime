.PHONY: help start run stop down restart status logs dev backend frontend tidy build test test-load proto routes

# Default target when running just 'make'
.DEFAULT_GOAL := start

help:
	@echo ""
	@echo "=========================================================="
	@echo "  🚕 Urban Prime Mobility OS - Commands"
	@echo "=========================================================="
	@echo ""
	@echo "  make start (or make)  - 🚀 Start entire project (all services + frontend)"
	@echo "  make stop             - 🛑 Stop entire project"
	@echo "  make restart          - 🔄 Restart entire project"
	@echo "  make status           - 📊 View container health & status"
	@echo "  make logs             - 📜 View live logs from all services"
	@echo "  make dev              - 💻 Local dev mode (Docker infra + Go backend + Next.js)"
	@echo "  make test             - 🧪 Run all unit and integration tests"
	@echo "  make tidy             - 🧹 Tidy all Go modules"
	@echo "  make build            - 🔨 Compile all Go binaries"
	@echo ""

# 1-Command: Start all services and frontend in Docker
start:
	@echo "🚀 Starting Urban Prime (all microservices + frontend)..."
	docker compose up -d --build
	@echo ""
	@echo "✅ All services and frontend are running!"
	@echo "   👉 Web App:        http://localhost:3000"
	@echo "   👉 Rider Portal:   http://localhost:3000/rider"
	@echo "   👉 Driver Cockpit: http://localhost:3000/driver"
	@echo "   👉 APISIX Gateway: http://localhost:9080"
	@echo "   👉 Jaeger Tracing: http://localhost:16686"

run: start

# 1-Command: Stop everything
stop:
	@echo "🛑 Stopping all Urban Prime containers..."
	docker compose down

down: stop

# Restart everything
restart:
	@echo "🔄 Restarting all Urban Prime containers..."
	docker compose restart

# View running container status
status:
	docker compose ps

# Tail live logs
logs:
	docker compose logs -f

# Local development mode: Run infra in Docker, Go services + Next.js with hot-reload
dev:
	@echo "⚡ Starting Docker infrastructure..."
	docker compose up -d postgres redis kafka centrifugo apisix jaeger
	@echo "⚡ Launching Go Microservices..."
	go run ./devserver/main.go

# Run frontend in local dev mode (npm run dev)
frontend:
	cd frontend && npm run dev

# Run all 6 Go microservices on host
backend:
	go run ./devserver/main.go

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
	go work sync

# Build all Go service binaries into /bin
build: tidy
	cd Services/auth-service && go build -o bin/auth-service ./cmd
	cd Services/trip-service && go build -o bin/trip-service ./cmd
	cd Services/driver-service && go build -o bin/driver-service ./cmd
	cd Services/location-service && go build -o bin/location-service ./cmd
	cd Services/payment-service && go build -o bin/payment-service ./cmd
	cd Services/notification-service && go build -o bin/notification-service ./cmd

# Run tests
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

proto:
	protoc --go_out=proto/gen --go_opt=paths=source_relative \
	       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
	       -I=proto proto/auth/v1/auth.proto proto/trip/v1/trip.proto \
	       proto/driver/v1/driver.proto proto/location/v1/location.proto \
	       proto/payment/v1/payment.proto
