package config

import (
	"os"
	"testing"
)

func TestLoadHealthDefaults(t *testing.T) {
	t.Setenv("AUTH_SERVICE_HTTP_PORT", "")
	os.Unsetenv("AUTH_SERVICE_HTTP_PORT")
	cfg := Load()
	if cfg.HTTPPort != "8080" {
		t.Fatalf("http port %s", cfg.HTTPPort)
	}
	if cfg.GRPCPort != "50056" {
		t.Fatalf("grpc port %s", cfg.GRPCPort)
	}
	if cfg.JWTAccessTTLMin <= 0 || cfg.JWTRefreshTTLDays <= 0 {
		t.Fatal("token TTLs must be positive")
	}
}
