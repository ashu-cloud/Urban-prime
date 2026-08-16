# Real-Time Cab Booking Platform — Architecture & Design

A production-grade, distributed microservices platform built in **Go 1.22**, utilizing gRPC for synchronous inter-service communications, Kafka for event streaming, Redis Geo for driver spatial indexing, Apache APISIX for API Gateway routing, Centrifugo for WebSocket real-time updates, and Stripe for payments.

---

## System Topology & Communication Flow

```mermaid
graph TD
    Rider[Rider Web/Mobile] -->|HTTP / WS| Gateway[Apache APISIX Gateway :9080]
    Driver[Driver Web/Mobile] -->|HTTP / WS| Gateway

    Gateway -->|gRPC| TripSvc[Trip Service :50051]
    Gateway -->|gRPC| DriverSvc[Driver Service :50052]
    Gateway -->|gRPC| LocationSvc[Location Service :50053]
    Gateway -->|gRPC| PaymentSvc[Payment Service :50054]

    LocationSvc -->|Fast Writes| RedisGeo[(Redis Geo :6379)]
    LocationSvc -->|Location Stream| Kafka{Kafka Cluster :9092}

    TripSvc -->|Trip State & Saga| Postgres[(PostgreSQL 16 :5432)]
    TripSvc -->|OSRM Routing| OSRM[OSRM Engine]
    TripSvc -->|Events| Kafka

    DriverSvc -->|Matchmaking & Geo Query| RedisGeo
    DriverSvc -->|Driver Profiles| Postgres
    DriverSvc -->|Events| Kafka

    PaymentSvc -->|Stripe API| Stripe[Stripe Payments]
    PaymentSvc -->|Events| Kafka

    Kafka -->|Notification Events| NotifSvc[Notification Service :50055]
    NotifSvc -->|WS Push| Centrifugo[Centrifugo Engine :8000]
    Centrifugo -->|Real-Time WS| Rider
    Centrifugo -->|Real-Time WS| Driver
```

---

## Infrastructure Ports & Datastores

| Component | Port | Description |
| :--- | :--- | :--- |
| **Apache APISIX** | `9080` (Proxy), `9090` (Control) | API Gateway (JWT Auth, Rate Limiting, Route Proxying) |
| **PostgreSQL 16** | `5432` | Relational database for Trips, Drivers, Payments |
| **Redis 7** | `6379` | Geo-spatial index for driver locations & session store |
| **Apache Kafka** | `9092` / `29092` | Asynchronous event bus |
| **Centrifugo v5** | `8000` | Real-time WebSocket connection engine |
| **Jaeger OTel** | `16686` (UI), `4317` (OTLP gRPC) | OpenTelemetry distributed tracing |
| **Trip Service** | `50051` | gRPC service (Saga Orchestrator & State Machine) |
| **Driver Service** | `50052` | gRPC service (Matchmaking algorithm) |
| **Location Service** | `50053` | gRPC service (GPS Ingestion) |
| **Payment Service** | `50054` | gRPC service (Stripe payment holds & refunds) |
| **Notification Service** | `50055` | gRPC service (Centrifugo WS bridge) |
