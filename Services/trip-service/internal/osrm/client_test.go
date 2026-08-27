package osrm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cab-booking/trip-service/internal/domain"
)

func TestGetRouteUsesOSRMResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "Ok",
			"routes": []map[string]any{
				{"distance": 5420.0, "duration": 720.0},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	res, err := client.GetRoute(context.Background(),
		domain.Location{Latitude: 12.97, Longitude: 77.59},
		domain.Location{Latitude: 12.98, Longitude: 77.60},
	)
	if err != nil {
		t.Fatal(err)
	}
	if res.DistanceKm < 5.4 || res.DistanceKm > 5.5 {
		t.Fatalf("distance %v", res.DistanceKm)
	}
	if res.DurationSecs != 720 {
		t.Fatalf("duration %d", res.DurationSecs)
	}
}

func TestGetRouteFallsBackOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	res, err := client.GetRoute(context.Background(),
		domain.Location{Latitude: 12.97, Longitude: 77.59},
		domain.Location{Latitude: 13.07, Longitude: 77.69},
	)
	if err != nil {
		t.Fatal(err)
	}
	if res.DistanceKm <= 0 {
		t.Fatal("fallback must produce a positive distance")
	}
}

func TestHaversineUsesSquareRoot(t *testing.T) {
	pickup := domain.Location{Latitude: 12.0, Longitude: 77.0}
	dropoff := domain.Location{Latitude: 12.1, Longitude: 77.0}
	res := fallbackHaversineRoute(pickup, dropoff)
	// 0.1 deg latitude * 111km ≈ 11.1km, not 11.1^2.
	if res.DistanceKm < 10 || res.DistanceKm > 13 {
		t.Fatalf("haversine distance %v looks like missing sqrt", res.DistanceKm)
	}
}

func TestHaversineMinimumDistance(t *testing.T) {
	loc := domain.Location{Latitude: 12.97, Longitude: 77.59}
	res := fallbackHaversineRoute(loc, loc)
	if res.DistanceKm != 0.5 {
		t.Fatalf("min distance %v", res.DistanceKm)
	}
}
