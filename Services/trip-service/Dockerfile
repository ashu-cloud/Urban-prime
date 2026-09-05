# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY Services/trip-service/ ./Services/trip-service/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /bin/trip-service ./Services/trip-service/cmd/main.go

# ─── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/trip-service .
COPY Services/trip-service/migrations/ ./migrations/

EXPOSE 8051 50051

CMD ["./trip-service"]
