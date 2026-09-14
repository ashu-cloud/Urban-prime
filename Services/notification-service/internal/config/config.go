package config

import (
	"os"
	"strings"
)

// Config holds all configuration for the Notification Service (Centrifugo WebSocket gateway)
type Config struct {
	// RedisAddr — Redis server address, used for both stream consumption and (optionally) caching
	RedisAddr string
	// ConsumerGroupID — consumer group ID, allows multiple instances to share the load
	ConsumerGroupID string
	// CentrifugoURL — base URL of the Centrifugo server HTTP API
	CentrifugoURL string
	// CentrifugoAPIKey — shared secret for Centrifugo server-to-server HTTP publish API
	CentrifugoAPIKey string
	// Port — HTTP port for health checks
	Port string
}

// Load reads configuration from environment variables with local dev defaults
func Load() *Config {
	return &Config{
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		ConsumerGroupID:  getEnv("CONSUMER_GROUP_ID", "notification-service"),
		CentrifugoURL:    getEnv("CENTRIFUGO_URL", "http://localhost:8000"),
		CentrifugoAPIKey: getEnv("CENTRIFUGO_API_KEY", ""),
		Port:             getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultVal
}
