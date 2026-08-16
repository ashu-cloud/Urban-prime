package osrm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cab-booking/trip-service/internal/domain"
)

// Client is an HTTP client for making REST API calls to the OSRM (Open Source Routing Machine) engine.
type Client struct {
	baseURL    string       // Host URL (e.g., "http://router.project-osrm.org")
	httpClient *http.Client // Go standard library HTTP client
}

// NewClient constructs an OSRM API Client with a configured 10-second HTTP request timeout
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Prevents hanging requests if OSRM service is slow
		},
	}
}

// osrmResponse matches the JSON payload structure returned by OSRM API REST endpoint.
// In Go, `json:"..."` struct tags tell `json.NewDecoder` how to map JSON keys to Go struct fields.
type osrmResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance float64 `json:"distance"` // Meters (e.g. 5420.5 meters)
		Duration float64 `json:"duration"` // Seconds (e.g. 720.0 seconds)
	} `json:"routes"`
}

// RouteResult holds parsed distance and duration values ready for fare pricing calculation
type RouteResult struct {
	DistanceKm   float64
	DurationSecs int64
}

// GetRoute fetches driving distance and duration from OSRM API for pickup & dropoff coordinates
func (c *Client) GetRoute(ctx context.Context, pickup, dropoff domain.Location) (*RouteResult, error) {
	// Construct OSRM endpoint URL format: /route/v1/driving/{lng1},{lat1};{lng2},{lat2}?overview=false
	// NOTE: OSRM expects Longitude FIRST, then Latitude!
	url := fmt.Sprintf(
		"%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=false",
		c.baseURL,
		pickup.Longitude, pickup.Latitude,
		dropoff.Longitude, dropoff.Latitude,
	)

	// Create HTTP GET request attached to context (allows cancellation if client disconnects)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSRM request: %w", err)
	}

	// Send HTTP request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// FALLBACK: If public OSRM server is down or unreachable, use Haversine geometry calculation
		return fallbackHaversineRoute(pickup, dropoff), nil
	}
	// 'defer' guarantees response body is closed when function returns, avoiding memory leaks
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackHaversineRoute(pickup, dropoff), nil
	}

	// Decode JSON response stream into Go struct
	var osrmResp osrmResponse
	if err := json.NewDecoder(resp.Body).Decode(&osrmResp); err != nil {
		return fallbackHaversineRoute(pickup, dropoff), nil
	}

	if len(osrmResp.Routes) == 0 {
		return fallbackHaversineRoute(pickup, dropoff), nil
	}

	route := osrmResp.Routes[0]
	return &RouteResult{
		DistanceKm:   route.Distance / 1000.0, // Convert meters to kilometers
		DurationSecs: int64(route.Duration),
	}, nil
}

// fallbackHaversineRoute is a fault-tolerant geometric distance estimator.
// If external OSRM API is temporarily down, this prevents system crash and allows trip requests to continue smoothly.
func fallbackHaversineRoute(pickup, dropoff domain.Location) *RouteResult {
	// Approximate distance calculation using latitude/longitude delta
	latDiff := (dropoff.Latitude - pickup.Latitude) * 111.0
	lngDiff := (dropoff.Longitude - pickup.Longitude) * 111.0 * 0.85
	distKm := (latDiff*latDiff + lngDiff*lngDiff)
	if distKm < 0.5 {
		distKm = 0.5
	}
	durationMins := distKm * 3.0 // Assume average speed ~20 km/h in urban traffic
	return &RouteResult{
		DistanceKm:   distKm,
		DurationSecs: int64(durationMins * 60),
	}
}
