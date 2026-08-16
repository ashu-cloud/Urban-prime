package config

import "os"

// Config holds all environment-driven configuration for the Location Service
type Config struct {
	// Port is the gRPC server port — default is 50053
	Port string
	// RedisAddr is the Redis server address — used for GEOADD driver positions
	RedisAddr string
	// KafkaBrokers is a comma-separated list of Kafka broker addresses
	KafkaBrokers string
}

// Load reads configuration from environment variables with sensible local development defaults
func Load() *Config {
	return &Config{
		Port:         getEnv("LOCATION_SERVICE_PORT", "50053"),
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
