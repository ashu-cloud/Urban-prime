# Auth Service + APISIX Gateway + Kafka Wiring — Implementation Plan

> **Status**: AWAITING USER APPROVAL — No code will be written until approved.

---

## Overview

This is the step that makes the platform production-ready. Three interconnected concerns:

1. **Auth Service** — Issues JWT tokens for riders and drivers. Every request carries a JWT. No JWT = 401.
2. **APISIX Gateway** — Validates JWTs before they ever touch a microservice. Also handles routing, rate limiting, and CORS. The single entry point for all external traffic.
3. **Kafka Wiring** — Right now each service publishes events but nobody fully consumes them correctly. We close the loop: events trigger real actions across services (e.g. `MatchAccepted` in Kafka → Trip Service updates trip to `ASSIGNED`).

---

## Open Questions (Please Answer Before I Implement)

> [!IMPORTANT]
> **Q1 — Auth users table**: Should Riders and Drivers share one `users` table with a `role` column (`RIDER`, `DRIVER`, `ADMIN`), or should they be in separate tables?
> **Recommendation**: Single `users` table with `role` column — simpler to manage JWTs with a single `sub` + `role` claim.

> [!IMPORTANT]
> **Q2 — Password hashing**: Use `bcrypt` (industry standard, slower but secure) or `argon2id` (modern best practice but requires CGO)?
> **Recommendation**: `bcrypt` — no CGO dependency, works cleanly in all Go environments.

> [!IMPORTANT]
> **Q3 — Token refresh strategy**: Short-lived access token (15 min) + long-lived refresh token (7 days), or single long-lived token (7 days)?
> **Recommendation**: Access + Refresh pair — industry standard, more secure. Refresh token stored in PostgreSQL.

> [!NOTE]
> **Q4 — Auth Service port**: Suggest `:50056` (gRPC) + `:8080` (HTTP REST for `/login`, `/register`, `/refresh`). Auth needs HTTP endpoints since browser/mobile clients call it directly (gRPC is for internal service-to-service use).

---

## Part 1 — Auth Service

### What It Does

| Endpoint | Method | Who Calls It | What It Returns |
| :--- | :--- | :--- | :--- |
| `POST /auth/register` | HTTP | Rider/Driver app | Creates account, returns JWT pair |
| `POST /auth/login` | HTTP | Rider/Driver app | Validates credentials, returns JWT pair |
| `POST /auth/refresh` | HTTP | Any client | Exchanges refresh token for new access token |
| `POST /auth/logout` | HTTP | Any client | Invalidates refresh token |
| `ValidateToken` | gRPC | APISIX (internal) | Validates JWT, returns decoded claims |

### JWT Claims Structure

```json
{
  "sub": "user-uuid-here",
  "role": "RIDER",
  "email": "user@example.com",
  "iat": 1724000000,
  "exp": 1724000900
}
```

### Database Schema — `users` Table

```sql
CREATE TABLE users (
    user_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    phone         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'RIDER',  -- RIDER | DRIVER | ADMIN
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refresh_tokens (
    token_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,   -- bcrypt hash of refresh token
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Package Structure

```
Services/auth-service/
├── cmd/
│   └── main.go                 ← HTTP + gRPC server bootstrap
├── migrations/
│   ├── 000001_create_users.up.sql
│   └── 000001_create_users.down.sql
├── internal/
│   ├── config/
│   │   └── config.go           ← JWT secret, DB DSN, token TTLs, ports
│   ├── domain/
│   │   └── user.go             ← User struct, Role enum, token types
│   ├── repository/
│   │   └── user_repo.go        ← Raw SQL: CreateUser, GetByEmail, StoreRefreshToken
│   ├── jwt/
│   │   └── manager.go          ← Issue/Validate/Refresh JWT tokens (golang-jwt/jwt/v5)
│   ├── handler/
│   │   ├── http_handler.go     ← net/http handlers: /register, /login, /refresh, /logout
│   │   └── grpc_handler.go     ← gRPC handler: ValidateToken (called by APISIX plugin)
│   └── middleware/
│       └── cors.go             ← CORS headers for browser clients
└── go.mod
```

### Dependencies

| Package | Purpose |
| :--- | :--- |
| `github.com/golang-jwt/jwt/v5` | JWT creation and validation |
| `golang.org/x/crypto/bcrypt` | Password hashing |
| `github.com/jackc/pgx/v5` | PostgreSQL |
| `github.com/golang-migrate/migrate/v4` | Auto-run SQL migrations |
| `github.com/google/uuid` | UUID generation |
| `google.golang.org/grpc` | gRPC for ValidateToken (internal) |

---

## Part 2 — APISIX Gateway Wiring

### Current State

APISIX is running (`deploy/apisix/config.yaml`) but has no routes configured — it just proxies everything blindly. We need to configure it properly via the **Admin API** at startup.

### Route Configuration Plan

APISIX routes are created via its REST Admin API (`http://localhost:9090`). We will create a **setup script** (`deploy/apisix/setup_routes.sh`) that runs `curl` commands to register all routes with JWT validation.

| Route | Upstream Service | JWT Required | Rate Limit |
| :--- | :--- | :--- | :--- |
| `POST /auth/register` | Auth Service `:8080` | ❌ No | 10 req/min |
| `POST /auth/login` | Auth Service `:8080` | ❌ No | 10 req/min |
| `POST /auth/refresh` | Auth Service `:8080` | ❌ No | 20 req/min |
| `POST /v1/trips` | Trip Service `:50051` (via grpc-transcode) | ✅ Yes | 30 req/min |
| `GET /v1/trips/:id` | Trip Service `:50051` | ✅ Yes | 60 req/min |
| `PUT /v1/drivers/status` | Driver Service `:50052` | ✅ Yes | 60 req/min |
| `PUT /v1/drivers/location` | Location Service `:50053` | ✅ Yes | 200 req/min (high!) |
| `GET /v1/drivers/:id` | Driver Service `:50052` | ✅ Yes | 60 req/min |

> [!NOTE]
> **gRPC transcoding note**: APISIX has a built-in `grpc-transcode` plugin that converts HTTP/1.1 JSON requests into gRPC calls automatically. This means mobile/browser apps send normal REST JSON, and APISIX converts it to gRPC internally. No gRPC client needed in the frontend.

### JWT Validation Flow

```
Rider App → POST /v1/trips  (Authorization: Bearer <JWT>)
    │
    ▼
APISIX Gateway
    ├── jwt-auth plugin validates signature + expiry
    ├── extracts { sub, role } from claims
    ├── injects X-User-ID: <sub> and X-User-Role: <role> headers
    └── forwards to Trip Service (with injected headers, never raw JWT)

Trip Service
    └── reads X-User-ID from header → that's the rider_id!
        (Never trusts client-supplied rider_id in request body)
```

### Files to Create/Modify

#### [NEW] `deploy/apisix/setup_routes.sh`
Shell script using `curl` to register all routes via APISIX Admin API (run once after `make dev-up`).

#### [MODIFY] `deploy/apisix/config.yaml`
Add `grpc-transcode` to the plugins list.

#### [MODIFY] `deploy/docker-compose.yml`
Add the `auth-service` container.

---

## Part 3 — Kafka Wiring (Closing the Event Loop)

### Current Problem

Right now services **publish** Kafka events, but nothing **consumes** them to trigger real actions:
- Trip Service publishes `trip.events.v1 { status: MATCHING }` → but Driver Service never reads this to start dispatching
- Driver Service publishes `driver.match.v1 { ACCEPTED }` → but Trip Service never reads this to update trip status to `ASSIGNED`

This is the gap. Without closing this loop, the Saga is broken.

### Event Flow to Wire Up

```
[1] Trip Service publishes → trip.events.v1 { status: MATCHING }
        │
        ▼ Consumer: Driver Service
    Driver Service reads MATCHING event
    → Calls its own DispatchLoop.FindAndDispatchDriver()
    → On success: publishes driver.match.v1 { ACCEPTED, driver_id }

[2] Driver Service publishes → driver.match.v1 { ACCEPTED }
        │
        ▼ Consumer: Trip Service
    Trip Service reads ACCEPTED event
    → Updates trip status: MATCHING → ASSIGNED
    → Sets trip.driver_id = driver_id
    → Publishes trip.events.v1 { status: ASSIGNED }

[3] Trip Service publishes → trip.events.v1 { status: ASSIGNED }
        │
        ▼ Consumer: Notification Service (already wired)
    Pushes "Driver matched!" to rider via Centrifugo WebSocket
```

### New Kafka Consumer Files

#### Driver Service — New Consumer
```
Services/driver-service/internal/kafka/consumer.go
```
- Consumes `trip.events.v1` topic
- Consumer group: `driver-service`
- Triggers dispatch loop when `status == "MATCHING"`

#### Trip Service — New Consumer
```
Services/trip-service/internal/kafka/consumer.go
```
- Consumes `driver.match.v1` topic
- Consumer group: `trip-service`
- Updates trip to `ASSIGNED` when event type is `ACCEPTED`
- Triggers compensation saga when event type is `EXHAUSTED`

---

## Files Summary

### [NEW] Auth Service
- `Services/auth-service/cmd/main.go`
- `Services/auth-service/go.mod`
- `Services/auth-service/migrations/000001_create_users.up.sql`
- `Services/auth-service/migrations/000001_create_users.down.sql`
- `Services/auth-service/internal/config/config.go`
- `Services/auth-service/internal/domain/user.go`
- `Services/auth-service/internal/repository/user_repo.go`
- `Services/auth-service/internal/jwt/manager.go`
- `Services/auth-service/internal/handler/http_handler.go`
- `Services/auth-service/internal/handler/grpc_handler.go`

### [NEW] APISIX Configuration
- `deploy/apisix/setup_routes.sh`

### [MODIFY] Existing Files
- `deploy/apisix/config.yaml` — add `grpc-transcode` plugin
- `deploy/docker-compose.yml` — add auth-service container
- `go.work` — add `./Services/auth-service`
- `Makefile` — add auth-service to `build` and `tidy` targets
- `.env.example` — add `AUTH_SERVICE_PORT`, `JWT_SECRET`, `JWT_ACCESS_TTL_MIN`, `JWT_REFRESH_TTL_DAYS`
- `Services/driver-service/cmd/main.go` — start Kafka consumer goroutine
- `Services/driver-service/internal/kafka/consumer.go` — NEW: trip.events.v1 consumer
- `Services/trip-service/cmd/main.go` — start Kafka consumer goroutine
- `Services/trip-service/internal/kafka/consumer.go` — NEW: driver.match.v1 consumer

---

## Verification Plan

### Auth Service
1. `go build ./Services/auth-service/cmd/main.go` — must compile clean
2. `POST /auth/register` → returns `{ access_token, refresh_token }`
3. `POST /auth/login` → returns JWT pair
4. `POST /auth/refresh` with expired access token → returns new access token
5. Verify JWT claims contain `sub`, `role`, `email`

### APISIX
1. `POST /v1/trips` without JWT → APISIX returns 401 (never reaches Trip Service)
2. `POST /v1/trips` with valid JWT → APISIX forwards with `X-User-ID` header injected
3. `POST /auth/login` → passes through without JWT check

### Kafka Loop
1. Create trip → verify Driver Service logs show "Dispatch Loop Initiated"
2. Driver accepts → verify Trip Service logs show "Trip status updated to ASSIGNED"
3. No drivers available → verify Trip Service logs show "Saga compensation triggered"

---

## What's NOT in this plan (defer to later)
- **OAuth2 / Google Sign-In** — out of scope for now
- **Role-based authorization inside services** — APISIX enforces role at gateway level for now
- **Token blacklisting on logout** (via Redis) — deferred; current implementation revokes in PostgreSQL
