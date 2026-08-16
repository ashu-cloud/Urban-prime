# Location Service + Notification Service (Centrifugo) — Implementation Plan

## What We're Building

This is the feature that makes the app *look like Uber*. Two complementary services:

**Location Service** — The GPS Firehose
- Receives continuous GPS pings from the driver's mobile app (~1 ping every 3 seconds)
- Writes driver coordinates to **Redis Geo** (instantly available for dispatch matching)
- Publishes `DriverLocationUpdated` events to **Kafka** (topic: `driver.location.v1`)
- Handles 100,000+ concurrent drivers → ~33,000 GPS pings/second

**Notification Service** — The Centrifugo WebSocket Gateway
- Consumes events from **Kafka** (trip events, location updates, match events)
- Pushes real-time updates to rider and driver mobile/web apps via **Centrifugo HTTP API**
- The rider's map updates every 3 seconds showing the driver moving toward them
- No WebSocket connection management in Go — Centrifugo handles all 100k client connections

---

## How the Real-Time Flow Works End-to-End

```
Driver App (GPS ping every 3s)
    │
    ▼
Location Service (gRPC :50053)
    ├── GEOADD → Redis Geo [instant, ~1ms]
    │   └── Driver Service Dispatch Loop reads this for nearest-driver search
    └── Publish → Kafka topic `driver.location.v1` [async]
            │
            ▼
    Notification Service (Kafka Consumer)
            │
            └── POST → Centrifugo HTTP API
                    │   /api/publish channel="trip#<trip_id>" 
                    │   payload: { driver_lat, driver_lng, driver_id }
                    │
                    ▼
            Centrifugo Engine (WebSocket Server :8000)
                    │
                    └── WebSocket push → Rider's browser/app
                            └── Map marker updates with driver's live position! 🚖
```

---

## Architecture Decisions

### Why Two Separate Services?
- **Separation of Concerns**: Location ingestion is CPU-bound (high-throughput writes). WebSocket broadcasting is I/O-bound (many connections).
- **Independent Scaling**: You can scale the Location Service to 10 pods during peak hours and the Notification Service to 5 pods, independently.
- **Fault Isolation**: If Centrifugo goes down, Location Service continues writing to Redis & Kafka. Location data is never lost.

### Why Not Write Directly to Centrifugo from Location Service?
- Centrifugo would become a single point of failure for GPS ingestion.
- Kafka provides a durable buffer: if the Notification Service restarts, it replays events.

---

## Centrifugo Configuration Updates Needed

Current `config.json` is missing:
- `history_size` and `history_ttl` — enables "catch-up" for reconnecting clients
- A `tracking` namespace for driver location channels
- `api_key` used for server-to-server HTTP publish API

Updated `deploy/centrifugo/config.json`:
```json
{
  "token_hmac_secret_key": "my_secret_token_hmac_key",
  "api_key": "my_secret_api_key",
  "admin": true,
  "admin_password": "admin",
  "admin_secret": "my_admin_secret",
  "namespaces": [
    { "name": "trip", "history_size": 10, "history_ttl": "60s" },
    { "name": "tracking", "history_size": 5, "history_ttl": "30s" }
  ]
}
```

Channel naming convention:
- `trip#<trip_id>` — Trip status updates (accepted, started, completed)
- `tracking#<trip_id>` — Live driver GPS coordinates during an active trip

---

## Package Structure

### Location Service

```
Services/location-service/
├── cmd/
│   └── main.go               ← gRPC server bootstrap
├── internal/
│   ├── config/
│   │   └── config.go         ← Port, Redis addr, Kafka brokers
│   ├── geo/
│   │   └── redis_geo.go      ← GEOADD to Redis (shared pattern from driver-service)
│   ├── kafka/
│   │   └── producer.go       ← Publishes LocationUpdatedEvent to driver.location.v1
│   └── handler/
│       └── location_handler.go ← gRPC handler: UpdateDriverLocation
└── go.mod
```

### Notification Service

```
Services/notification-service/
├── cmd/
│   └── main.go               ← Kafka consumer start + gRPC (future)
├── internal/
│   ├── config/
│   │   └── config.go         ← Kafka brokers, Centrifugo API URL, API Key
│   ├── centrifugo/
│   │   └── client.go         ← HTTP client for Centrifugo publish API
│   └── consumer/
│       └── kafka_consumer.go ← Kafka consumer group: driver.location.v1 + trip.events.v1
└── go.mod
```

---

## Key API: Centrifugo HTTP Publish

Centrifugo exposes a simple HTTP API. To push a real-time update to all subscribers of a channel:

```http
POST http://centrifugo:8000/api/publish
Authorization: apikey my_secret_api_key
Content-Type: application/json

{
  "channel": "tracking#trip-abc-123",
  "data": {
    "driver_id": "driver-xyz",
    "latitude": 28.6139,
    "longitude": 77.2090,
    "timestamp": "2026-08-16T20:00:00Z"
  }
}
```

The rider's browser subscribes to `tracking#<trip_id>` and their map marker moves automatically!

---

## Proto Definition: Location Service

```protobuf
service LocationService {
  rpc UpdateDriverLocation(UpdateDriverLocationRequest) returns (UpdateDriverLocationResponse);
  rpc GetDriverLocation(GetDriverLocationRequest) returns (GetDriverLocationResponse);
}
```

---

## External Dependencies

| Service | Package | Purpose |
| :--- | :--- | :--- |
| **Location Service** | `github.com/redis/go-redis/v9` | GEOADD driver location |
| **Location Service** | `github.com/twmb/franz-go` | Kafka location event producer |
| **Location Service** | `google.golang.org/grpc` | gRPC server |
| **Notification Service** | `github.com/twmb/franz-go` | Kafka consumer group |
| **Notification Service** | `net/http` (stdlib) | HTTP POST to Centrifugo API |

---

## Verification Plan

1. `go build ./Services/location-service/cmd/main.go` — must compile clean
2. `go build ./Services/notification-service/cmd/main.go` — must compile clean
3. Start infra: `docker compose -f deploy/docker-compose.yml up -d`
4. Test with grpcurl: send `UpdateDriverLocation(driver_id, lat, lng)` 
5. Verify Redis: `redis-cli GEOPOS drivers:available <driver_id>` shows coordinates
6. Verify Kafka: `kafkacat -C -b localhost:9092 -t driver.location.v1`
7. Open Centrifugo admin UI at `http://localhost:8000`
8. Verify location events appear in the Centrifugo channel `tracking#test-trip`
