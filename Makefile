.PHONY: help dev-up dev-down dev-status dev-logs proto-gen build tidy

help:
	@echo "Available commands:"
	@echo "  make dev-up     - Start all infrastructure containers (Postgres, Redis, Kafka, Centrifugo, APISIX, Jaeger)"
	@echo "  make dev-down   - Stop all infrastructure containers"
	@echo "  make dev-status - Show status of infrastructure containers"
	@echo "  make dev-logs   - Tail logs of infrastructure containers"
	@echo "  make proto-gen  - Generate Go code from Protobuf definitions"
	@echo "  make tidy       - Run go mod tidy across all modules"
	@echo "  make build      - Build all Go microservices"

dev-up:
	docker-compose -f deploy/docker-compose.yml up -d

dev-down:
	docker-compose -f deploy/docker-compose.yml down

dev-status:
	docker-compose -f deploy/docker-compose.yml ps

dev-logs:
	docker-compose -f deploy/docker-compose.yml logs -f

tidy:
	cd pkg && go mod tidy
	cd Services/auth-service && go mod tidy
	cd Services/trip-service && go mod tidy
	cd Services/driver-service && go mod tidy
	cd Services/location-service && go mod tidy
	cd Services/payment-service && go mod tidy
	cd Services/notification-service && go mod tidy

build: tidy
	cd Services/auth-service && go build -o bin/auth-service ./cmd
	cd Services/trip-service && go build -o bin/trip-service ./cmd
	cd Services/driver-service && go build -o bin/driver-service ./cmd
	cd Services/location-service && go build -o bin/location-service ./cmd
	cd Services/payment-service && go build -o bin/payment-service ./cmd
	cd Services/notification-service && go build -o bin/notification-service ./cmd
