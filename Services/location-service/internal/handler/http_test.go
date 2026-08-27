package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocationHealthAndUpdateHTTP(t *testing.T) {
	h := NewLocationHandler(&MockGeoClient{}, &MockKafkaProducer{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health %d", rec.Code)
	}

	body := `{"driverId":"drv_1","latitude":12.97,"longitude":77.59,"heading":90}`
	upd := httptest.NewRequest(http.MethodPost, "/api/v1/location/driver", strings.NewReader(body))
	upd.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, upd)
	if rec.Code != http.StatusOK {
		t.Fatalf("update %d %s", rec.Code, rec.Body.String())
	}
}
