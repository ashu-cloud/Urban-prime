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
	port := firstNonEmpty(os.Getenv("PORT"), os.Getenv("PAYMENT_SERVICE_PORT"), "50054")
	dbURL := firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_DSN"), "")
	stripeKey := firstNonEmpty(os.Getenv("STRIPE_SECRET_KEY"), "")

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		StripeSecretKey: stripeKey,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
