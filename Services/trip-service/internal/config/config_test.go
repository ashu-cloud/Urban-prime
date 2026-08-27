package config

import "testing"

func TestTripServicePaymentAddrDefault(t *testing.T) {
	t.Setenv("PAYMENT_SERVICE_ADDR", "")
	cfg := Load()
	if cfg.PaymentServiceAddr != "localhost:50054" {
		t.Fatalf("payment addr %s", cfg.PaymentServiceAddr)
	}
	if cfg.BaseFareCents <= 0 || cfg.PerKmRateCents <= 0 {
		t.Fatal("fare rates must be positive")
	}
}
