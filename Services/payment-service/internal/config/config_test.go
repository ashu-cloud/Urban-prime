package config

import "testing"

func TestPaymentServiceDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("PAYMENT_SERVICE_PORT", "")
	cfg := Load()
	if cfg.Port != "50054" {
		t.Fatalf("payment default port %s want 50054", cfg.Port)
	}
}
