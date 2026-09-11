# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY Services/ ./Services/
COPY devserver/ ./devserver/
COPY tests/ ./tests/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /bin/notification-service ./Services/notification-service/cmd/main.go

# ─── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/notification-service .

EXPOSE 50055

CMD ["./notification-service"]
