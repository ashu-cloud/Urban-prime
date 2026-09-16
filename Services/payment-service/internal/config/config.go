package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	HTTPPort        string
	DatabaseURL     string
	StripeSecretKey string
}

func Load() *Config {
	port := firstNonEmpty(os.Getenv("PAYMENT_SERVICE_PORT"), os.Getenv("PORT"), "50054")
	httpPort := firstNonEmpty(os.Getenv("PAYMENT_SERVICE_HTTP_PORT"), "8054")
	dbURL := firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_DSN"), "")
	stripeKey := firstNonEmpty(os.Getenv("STRIPE_SECRET_KEY"), "")

	return &Config{
		Port:            port,
		HTTPPort:        httpPort,
		DatabaseURL:     dbURL,
		StripeSecretKey: stripeKey,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
