package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

type probe struct {
	name    string
	network string
	addr    string
}

func TestInfrastructureHealth(t *testing.T) {
	probes := []probe{
		{name: "postgres", network: "tcp", addr: envOr("POSTGRES_ADDR", "127.0.0.1:5432")},
		{name: "redis", network: "tcp", addr: envOr("REDIS_ADDR", "127.0.0.1:6379")},
		{name: "kafka", network: "tcp", addr: envOr("KAFKA_ADDR", "127.0.0.1:9092")},
		{name: "centrifugo", network: "tcp", addr: envOr("CENTRIFUGO_ADDR", "127.0.0.1:8000")},
		{name: "apisix", network: "tcp", addr: envOr("APISIX_ADDR", "127.0.0.1:9080")},
		{name: "auth-http", network: "tcp", addr: envOr("AUTH_HTTP_ADDR", "127.0.0.1:8080")},
		{name: "trip-grpc", network: "tcp", addr: envOr("TRIP_GRPC_ADDR", "127.0.0.1:50051")},
		{name: "driver-grpc", network: "tcp", addr: envOr("DRIVER_GRPC_ADDR", "127.0.0.1:50052")},
		{name: "location-grpc", network: "tcp", addr: envOr("LOCATION_GRPC_ADDR", "127.0.0.1:50053")},
		{name: "payment-grpc", network: "tcp", addr: envOr("PAYMENT_GRPC_ADDR", "127.0.0.1:50054")},
		{name: "auth-grpc", network: "tcp", addr: envOr("AUTH_GRPC_ADDR", "127.0.0.1:50056")},
		{name: "trip-http", network: "tcp", addr: envOr("TRIP_HTTP_ADDR", "127.0.0.1:8051")},
		{name: "driver-http", network: "tcp", addr: envOr("DRIVER_HTTP_ADDR", "127.0.0.1:8052")},
		{name: "location-http", network: "tcp", addr: envOr("LOCATION_HTTP_ADDR", "127.0.0.1:8053")},
	}

	up := 0
	for _, p := range probes {
		p := p
		t.Run(p.name, func(t *testing.T) {
			conn, err := net.DialTimeout(p.network, p.addr, 600*time.Millisecond)
			if err != nil {
				t.Skipf("%s not reachable at %s: %v", p.name, p.addr, err)
			}
			_ = conn.Close()
			up++
		})
	}
	t.Logf("reachable services: recorded via subtests")
	_ = up
}

func TestAuthHTTPHealthAndE2E(t *testing.T) {
	base := "http://" + envOr("AUTH_HTTP_ADDR", "127.0.0.1:8080")
	client := &http.Client{Timeout: 2 * time.Second}

	health, err := client.Get(base + "/health")
	if err != nil {
		t.Skipf("auth-service not running: %v", err)
	}
	defer health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status %d", health.StatusCode)
	}

	email := fmt.Sprintf("live-%d@example.com", time.Now().UnixNano())
	regBody := map[string]string{
		"email":     email,
		"phone":     fmt.Sprintf("+1555%06d", time.Now().UnixNano()%1000000),
		"password":  "LiveTest99",
		"full_name": "Live Tester",
		"role":      "RIDER",
	}
	reg, err := postJSON(client, base+"/auth/register", regBody)
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Body.Close()
	if reg.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(reg.Body)
		t.Fatalf("register %d %s", reg.StatusCode, raw)
	}

	login, err := postJSON(client, base+"/auth/login", map[string]string{
		"email":    email,
		"password": "LiveTest99",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer login.Body.Close()
	if login.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(login.Body)
		t.Fatalf("login %d %s", login.StatusCode, raw)
	}

	var payload map[string]any
	if err := json.NewDecoder(login.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	access, _ := payload["access_token"].(string)
	refresh, _ := payload["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("missing tokens: %+v", payload)
	}

	ref, err := postJSON(client, base+"/auth/refresh", map[string]string{"refresh_token": refresh})
	if err != nil {
		t.Fatal(err)
	}
	defer ref.Body.Close()
	if ref.StatusCode != http.StatusOK {
		t.Fatalf("refresh %d", ref.StatusCode)
	}

	bad, err := postJSON(client, base+"/auth/refresh", map[string]string{"refresh_token": access})
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Body.Close()
	if bad.StatusCode == http.StatusOK {
		t.Fatal("access token must not refresh a session")
	}
}

func TestCentrifugoHealth(t *testing.T) {
	url := "http://" + envOr("CENTRIFUGO_ADDR", "127.0.0.1:8000") + "/health"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("centrifugo not running: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Fatalf("centrifugo health %d", resp.StatusCode)
	}
}

func TestAPISIXGatewayHealth(t *testing.T) {
	url := "http://" + envOr("APISIX_ADDR", "127.0.0.1:9080")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("apisix not running: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Fatalf("apisix status %d", resp.StatusCode)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func postJSON(client *http.Client, url string, body any) (*http.Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}
