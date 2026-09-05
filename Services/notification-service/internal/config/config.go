package config

import "os"

// Config holds all configuration for the Notification Service (Centrifugo WebSocket gateway)
type Config struct {
	// KafkaBrokers — comma-separated list of Kafka brokers to consume location & trip events from
	KafkaBrokers string
	// KafkaGroupID — consumer group ID, allows multiple instances to share the load
	KafkaGroupID string
	// CentrifugoURL — base URL of the Centrifugo server HTTP API
	CentrifugoURL string
	// CentrifugoAPIKey — shared secret for Centrifugo server-to-server HTTP publish API
	CentrifugoAPIKey string
}

// Load reads configuration from environment variables with local dev defaults
func Load() *Config {
	return &Config{
		KafkaBrokers:     getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaGroupID:     getEnv("KAFKA_GROUP_ID", "notification-service"),
		CentrifugoURL:    getEnv("CENTRIFUGO_URL", "http://localhost:8000"),
		CentrifugoAPIKey: getEnv("CENTRIFUGO_API_KEY", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
