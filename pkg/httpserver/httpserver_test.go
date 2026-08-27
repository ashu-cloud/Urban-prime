package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	Health("auth-service")(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "auth-service") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if FirstNonEmpty("", "b", "c") != "b" {
		t.Fatal("expected b")
	}
}
