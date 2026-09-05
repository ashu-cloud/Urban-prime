# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Copy workspace files
COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY proto/ ./proto/

# Copy only auth-service source
COPY Services/auth-service/ ./Services/auth-service/

# Build the binary (statically linked, no CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /bin/auth-service ./Services/auth-service/cmd/main.go

# ─── Runtime Stage ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /bin/auth-service .

# Copy migrations so file://migrations resolves correctly at runtime
COPY Services/auth-service/migrations/ ./migrations/

EXPOSE 8080 50056

CMD ["./auth-service"]
