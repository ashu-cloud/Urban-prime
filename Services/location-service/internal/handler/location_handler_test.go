package handler

import (
	"context"
	"errors"
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
