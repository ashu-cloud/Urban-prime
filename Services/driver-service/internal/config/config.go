package config

import (
	"os"
)

// Config stores environment parameters for the Driver Service
type Config struct {
	Port         string
	DatabaseDSN  string
	RedisAddr    string
	KafkaBrokers string
}

// Load populates configuration from environment variables or sensible local defaults
func Load() *Config {
	return &Config{
		Port:         getEnv("DRIVER_SERVICE_PORT", "50052"),
		DatabaseDSN:  getEnv("POSTGRES_DSN", "postgres://cab_user:cab_password@localhost:5432/cab_booking_db?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
