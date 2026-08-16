package centrifugo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cab-booking/pkg/logger"
)

// Channel naming conventions used across the platform:
//   - tracking#<trip_id>  — live driver GPS coordinates during an active trip
//   - trip#<trip_id>      — trip lifecycle events (matched, started, completed, cancelled)
//
// The rider app subscribes to both channels using the trip_id as a namespace key.
// The driver app subscribes to trip#<trip_id> for status updates.

// Client is a lightweight HTTP client for the Centrifugo server-to-server publish API.
//
// HOW CENTRIFUGO WORKS (beginner explanation):
//   - Centrifugo is a standalone WebSocket server.
//   - Mobile/web clients connect to Centrifugo directly over WebSocket.
//   - Backend microservices (us) NEVER manage WebSocket connections.
//   - Instead, we POST messages to Centrifugo's HTTP API: POST /api/publish
//   - Centrifugo immediately fans out the message to all subscribed WebSocket clients.
//   - This decouples our Go services from WebSocket connection management entirely!
type Client struct {
	baseURL    string       // e.g. "http://centrifugo:8000"
	apiKey     string       // shared secret for HTTP publish API
	httpClient *http.Client // reused HTTP connection pool
}

// NewClient creates a Centrifugo HTTP API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// publishRequest is the JSON body sent to Centrifugo's /api/publish endpoint
type publishRequest struct {
	Channel string      `json:"channel"` // e.g. "tracking#trip-abc-123"
	Data    interface{} `json:"data"`    // any JSON-serialisable payload
}

// Publish sends a message to all WebSocket clients subscribed to the given channel.
//
// Example: Publish("tracking#trip-123", LocationPayload{...})
// → All rider browsers/apps subscribed to "tracking#trip-123" immediately receive the message.
// → Their map marker moves to the new driver position. This is the Uber live-tracking experience!
func (c *Client) Publish(ctx context.Context, channel string, data interface{}) error {
	payload := publishRequest{
		Channel: channel,
		Data:    data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("centrifugo: failed to marshal publish payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/publish", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("centrifugo: failed to create HTTP request: %w", err)
	}

	// Centrifugo server-to-server auth uses the api_key from config.json
	req.Header.Set("Authorization", fmt.Sprintf("apikey %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("centrifugo: HTTP publish request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("centrifugo: non-200 response: %d", resp.StatusCode)
	}

	return nil
}

// PublishDriverLocation is a typed helper for pushing a driver GPS update to the rider's map.
// It publishes to the channel: tracking#<trip_id>
func (c *Client) PublishDriverLocation(ctx context.Context, tripID, driverID string, lat, lng float64, bearing float32) {
	if tripID == "" {
		return // Only push live tracking when driver is on an active trip
	}

	channel := fmt.Sprintf("tracking#%s", tripID)
	payload := map[string]interface{}{
		"driver_id": driverID,
		"latitude":  lat,
		"longitude": lng,
		"bearing":   bearing,
		"timestamp": time.Now().UnixMilli(),
	}

	if err := c.Publish(ctx, channel, payload); err != nil {
		logger.Warn(ctx, "Centrifugo publish failed (non-fatal, map update missed one frame)",
			"channel", channel,
			"driver_id", driverID,
			"error", err,
		)
	}
}

// PublishTripEvent pushes a trip lifecycle event to the rider and driver apps.
// It publishes to the channel: trip#<trip_id>
//
// Examples of eventType: "DRIVER_MATCHED", "TRIP_STARTED", "TRIP_COMPLETED", "TRIP_CANCELLED"
func (c *Client) PublishTripEvent(ctx context.Context, tripID, eventType string, payload map[string]interface{}) {
	channel := fmt.Sprintf("trip#%s", tripID)

	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["event_type"] = eventType
	payload["timestamp"] = time.Now().UnixMilli()

	if err := c.Publish(ctx, channel, payload); err != nil {
		logger.Warn(ctx, "Centrifugo trip event publish failed",
			"channel", channel,
			"event_type", eventType,
			"error", err,
		)
	}
}
