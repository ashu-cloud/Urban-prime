package handler

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cab-booking/location-service/internal/kafka"
	locationv1 "github.com/cab-booking/proto/gen/location/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockGeoClient struct {
	UpdateDriverLocationFunc func(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error
	GetDriverLocationFunc    func(ctx context.Context, driverID string) (float64, float64, error)
}

func (m *MockGeoClient) UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error {
	if m.UpdateDriverLocationFunc != nil {
		return m.UpdateDriverLocationFunc(ctx, driverID, lat, lng, onTrip)
	}
	return nil
}

func (m *MockGeoClient) GetDriverLocation(ctx context.Context, driverID string) (float64, float64, error) {
	if m.GetDriverLocationFunc != nil {
		return m.GetDriverLocationFunc(ctx, driverID)
	}
	return 12.9, 77.5, nil
}

type MockKafkaProducer struct {
	PublishLocationUpdateFunc func(ctx context.Context, event kafka.LocationEvent) error
}

func (m *MockKafkaProducer) PublishLocationUpdate(ctx context.Context, event kafka.LocationEvent) error {
	if m.PublishLocationUpdateFunc != nil {
		return m.PublishLocationUpdateFunc(ctx, event)
	}
	return nil
}

func TestUpdateDriverLocation_Success(t *testing.T) {
	geoClient := &MockGeoClient{}
	prod := &MockKafkaProducer{}

	eventsPublished := 0
	prod.PublishLocationUpdateFunc = func(ctx context.Context, event kafka.LocationEvent) error {
		eventsPublished++
		return nil
	}

	handler := NewLocationHandler(geoClient, prod)

	req := &locationv1.UpdateDriverLocationRequest{
		DriverId:  "driver_1",
		Latitude:  12.9,
		Longitude: 77.5,
	}

	resp, err := handler.UpdateDriverLocation(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatal("Expected success response")
	}
	if eventsPublished != 1 {
		t.Errorf("Expected 1 Kafka event published, got %d", eventsPublished)
	}
}

func TestUpdateDriverLocation_MissingDriverID(t *testing.T) {
	handler := NewLocationHandler(&MockGeoClient{}, &MockKafkaProducer{})

	req := &locationv1.UpdateDriverLocationRequest{
		Latitude:  12.9,
		Longitude: 77.5,
	}

	_, err := handler.UpdateDriverLocation(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error due to missing driver ID")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument code, got %v", status.Code(err))
	}
}

func TestUpdateDriverLocation_RedisFailure_ContinuesToPublish(t *testing.T) {
	geoClient := &MockGeoClient{}
	prod := &MockKafkaProducer{}

	geoClient.UpdateDriverLocationFunc = func(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error {
		return errors.New("redis down")
	}

	eventsPublished := 0
	prod.PublishLocationUpdateFunc = func(ctx context.Context, event kafka.LocationEvent) error {
		eventsPublished++
		return nil
	}

	handler := NewLocationHandler(geoClient, prod)

	req := &locationv1.UpdateDriverLocationRequest{
		DriverId:  "driver_1",
		Latitude:  12.9,
		Longitude: 77.5,
	}

	resp, err := handler.UpdateDriverLocation(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error even if Redis fails, got %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatal("Expected success response")
	}
	if eventsPublished != 1 {
		t.Errorf("Expected Kafka event to still publish, got %d", eventsPublished)
	}
}

func TestUpdateDriverLocation_InvalidCoordinates(t *testing.T) {
	handler := NewLocationHandler(&MockGeoClient{}, &MockKafkaProducer{})
	cases := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"null island", 0, 0},
		{"lat too high", 91, 10},
		{"lat too low", -91, 10},
		{"lng too high", 10, 181},
		{"lng too low", 10, -181},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := handler.UpdateDriverLocation(context.Background(), &locationv1.UpdateDriverLocationRequest{
				DriverId:  "driver_1",
				Latitude:  tc.lat,
				Longitude: tc.lng,
			})
			if err == nil {
				t.Fatal("expected invalid argument")
			}
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code %v", status.Code(err))
			}
		})
	}
}

func TestGetDriverLocation(t *testing.T) {
	handler := NewLocationHandler(&MockGeoClient{}, &MockKafkaProducer{})
	resp, err := handler.GetDriverLocation(context.Background(), &locationv1.GetDriverLocationRequest{DriverId: "driver_1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Latitude == 0 && resp.Longitude == 0 {
		t.Fatal("expected stored coordinates")
	}

	_, err = handler.GetDriverLocation(context.Background(), &locationv1.GetDriverLocationRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code %v", status.Code(err))
	}
}

func TestGetDriverLocationNotFound(t *testing.T) {
	geo := &MockGeoClient{
		GetDriverLocationFunc: func(ctx context.Context, driverID string) (float64, float64, error) {
			return 0, 0, errors.New("missing")
		},
	}
	handler := NewLocationHandler(geo, &MockKafkaProducer{})
	_, err := handler.GetDriverLocation(context.Background(), &locationv1.GetDriverLocationRequest{DriverId: "ghost"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code %v", status.Code(err))
	}
}

func TestConcurrentLocationUpdates(t *testing.T) {
	var mu sync.Mutex
	count := 0
	geo := &MockGeoClient{
		UpdateDriverLocationFunc: func(ctx context.Context, driverID string, lat, lng float64, onTrip bool) error {
			mu.Lock()
			count++
			mu.Unlock()
			return nil
		},
	}
	handler := NewLocationHandler(geo, &MockKafkaProducer{})
	var wg sync.WaitGroup
	const n = 200
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := handler.UpdateDriverLocation(context.Background(), &locationv1.UpdateDriverLocationRequest{
				DriverId:  "driver_1",
				Latitude:  12.9,
				Longitude: 77.5 + float64(i)*0.0001,
			})
			if err != nil {
				t.Errorf("update: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if count != n {
		t.Fatalf("geo writes %d want %d", count, n)
	}
}
