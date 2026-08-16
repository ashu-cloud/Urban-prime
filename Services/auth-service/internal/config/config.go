package config

import (
	"os"
	"strconv"
)

// Config manages environment variables for Auth Service
type Config struct {
	HTTPPort          string
	GRPCPort          string
	DatabaseDSN       string
	JWTSecret         string
	JWTAccessTTLMin   int
	JWTRefreshTTLDays int
}

// Load populates configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		HTTPPort:          getEnv("AUTH_SERVICE_HTTP_PORT", "8080"),
		GRPCPort:          getEnv("AUTH_SERVICE_GRPC_PORT", "50056"),
		DatabaseDSN:       getEnv("POSTGRES_DSN", "postgres://cab_user:cab_password@localhost:5432/cab_booking_db?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "super_secret_jwt_signing_key_change_in_production"),
		JWTAccessTTLMin:   getEnvInt("JWT_ACCESS_TTL_MIN", 15),
		JWTRefreshTTLDays: getEnvInt("JWT_REFRESH_TTL_DAYS", 7),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
