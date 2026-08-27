package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	DatabaseDSN        string
	KafkaBrokers       string
	OSRMHost           string
	DriverServiceAddr  string
	PaymentServiceAddr string
	BaseFareCents      int64
	PerKmRateCents     int64
	PerMinRateCents    int64
	DefaultSurgeMult   float64
}

func Load() *Config {
	return &Config{
		Port:               getEnv("TRIP_SERVICE_PORT", "50051"),
		DatabaseDSN:        getEnv("POSTGRES_DSN", "postgres://cab_user:cab_password@localhost:5432/cab_booking_db?sslmode=disable"),
		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		OSRMHost:           getEnv("OSRM_HOST", "http://router.project-osrm.org"),
		DriverServiceAddr:  getEnv("DRIVER_SERVICE_ADDR", "localhost:50052"),
		PaymentServiceAddr: getEnv("PAYMENT_SERVICE_ADDR", "localhost:50054"),
		BaseFareCents:      getEnvInt64("BASE_FARE_CENTS", 3000),      // ₹30.00
		PerKmRateCents:     getEnvInt64("PER_KM_RATE_CENTS", 1500),    // ₹15.00/km
		PerMinRateCents:    getEnvInt64("PER_MIN_RATE_CENTS", 100),     // ₹1.00/min
		DefaultSurgeMult:   getEnvFloat64("DEFAULT_SURGE_MULT", 1.0),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvFloat64(key string, defaultVal float64) float64 {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			return val
		}
	}
	return defaultVal
}
