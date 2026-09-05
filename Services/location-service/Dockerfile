# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY Services/location-service/ ./Services/location-service/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /bin/location-service ./Services/location-service/cmd/main.go

# ─── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/location-service .

EXPOSE 8053 50053

CMD ["./location-service"]
