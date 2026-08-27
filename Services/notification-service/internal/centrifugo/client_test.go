package centrifugo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPublishSuccessAndAuthHeader(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "secret-key")
	if err := c.Publish(context.Background(), "trip#abc", map[string]string{"event": "ok"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "apikey secret-key" {
		t.Fatalf("auth header %q", gotAuth)
	}
	if gotBody["channel"] != "trip#abc" {
		t.Fatalf("channel %v", gotBody["channel"])
	}
}

func TestPublishNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "bad")
	if err := c.Publish(context.Background(), "trip#x", map[string]string{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishDriverLocationSkipsEmptyTrip(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "k")
	c.PublishDriverLocation(context.Background(), "", "drv", 1, 2, 90)
	if hits.Load() != 0 {
		t.Fatal("empty trip id must not publish")
	}
	c.PublishDriverLocation(context.Background(), "trip-1", "drv", 1, 2, 90)
	if hits.Load() != 1 {
		t.Fatalf("hits %d", hits.Load())
	}
}
