package config

import (
	"os"
)

type Config struct {
	Port            string
	DatabaseURL     string
	StripeSecretKey string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50055"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cab_user:cab_password@localhost:5432/cab_booking_db?sslmode=disable"
	}

	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		stripeKey = "sk_test_mock"
	}

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		StripeSecretKey: stripeKey,
	}
}
