package saga

import (
	"context"
	"errors"
	"testing"

	"github.com/cab-booking/trip-service/internal/config"
	"github.com/cab-booking/trip-service/internal/domain"
	"github.com/cab-booking/trip-service/internal/kafka"
	"github.com/cab-booking/trip-service/internal/osrm"
	"github.com/cab-booking/trip-service/internal/pricing"
)

// Mocks

type MockTripRepo struct {
	CreateFunc       func(ctx context.Context, trip *domain.Trip) error
	UpdateStatusFunc func(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error
	AssignDriverFunc func(ctx context.Context, tripID, driverID string, newStatus domain.TripStatus, step domain.SagaStepLog) error
}

func (m *MockTripRepo) Create(ctx context.Context, trip *domain.Trip) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, trip)
	}
	return nil
}
func (m *MockTripRepo) UpdateStatus(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, newStatus, step)
	}
	return nil
}
func (m *MockTripRepo) AssignDriver(ctx context.Context, tripID, driverID string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	if m.AssignDriverFunc != nil {
		return m.AssignDriverFunc(ctx, tripID, driverID, newStatus, step)
	}
	return nil
}

type MockOSRMClient struct {
	GetRouteFunc func(ctx context.Context, origin, destination domain.Location) (*osrm.RouteResult, error)
}

func (m *MockOSRMClient) GetRoute(ctx context.Context, origin, destination domain.Location) (*osrm.RouteResult, error) {
	if m.GetRouteFunc != nil {
		return m.GetRouteFunc(ctx, origin, destination)
	}
	return &osrm.RouteResult{DistanceKm: 10.0, DurationSecs: 600}, nil
}

type MockKafkaProducer struct {
	PublishTripEventFunc func(ctx context.Context, topic string, payload kafka.TripEventPayload) error
}

func (m *MockKafkaProducer) PublishTripEvent(ctx context.Context, topic string, payload kafka.TripEventPayload) error {
	if m.PublishTripEventFunc != nil {
		return m.PublishTripEventFunc(ctx, topic, payload)
	}
	return nil
}

type MockPaymentClient struct {
	AuthorizeHoldFunc func(ctx context.Context, tripID, riderID string, amountCents int64, currency, paymentMethodID string) (string, error)
	ReleaseHoldFunc   func(ctx context.Context, transactionID, tripID, reason string) error
	ReleaseHoldCalled bool
}

func (m *MockPaymentClient) AuthorizeHold(ctx context.Context, tripID, riderID string, amountCents int64, currency, paymentMethodID string) (string, error) {
	if m.AuthorizeHoldFunc != nil {
		return m.AuthorizeHoldFunc(ctx, tripID, riderID, amountCents, currency, paymentMethodID)
	}
	return "txn_123", nil
}
func (m *MockPaymentClient) ReleaseHold(ctx context.Context, transactionID, tripID, reason string) error {
	m.ReleaseHoldCalled = true
	if m.ReleaseHoldFunc != nil {
		return m.ReleaseHoldFunc(ctx, transactionID, tripID, reason)
	}
	return nil
}

func setupTestOrchestrator() (*Orchestrator, *MockTripRepo, *MockOSRMClient, *MockKafkaProducer, *MockPaymentClient) {
	repo := &MockTripRepo{}
	osrmClient := &MockOSRMClient{}
	producer := &MockKafkaProducer{}
	paymentClient := &MockPaymentClient{}
	calculator := pricing.NewCalculator(&config.Config{})

	orc := NewOrchestrator(repo, osrmClient, calculator, producer, paymentClient)
	return orc, repo, osrmClient, producer, paymentClient
}

func TestExecuteCreateTripSaga_Success(t *testing.T) {
	orc, _, _, producer, _ := setupTestOrchestrator()

	eventsPublished := 0
	producer.PublishTripEventFunc = func(ctx context.Context, topic string, payload kafka.TripEventPayload) error {
		eventsPublished++
		return nil
	}

	cmd := CreateTripCmd{
		RiderID:         "rider_1",
		Pickup:          domain.Location{Latitude: 12.9, Longitude: 77.5},
		Dropoff:         domain.Location{Latitude: 12.95, Longitude: 77.55},
		VehicleType:     "SEDAN",
		PaymentMethodID: "pm_card",
	}

	trip, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if trip == nil {
		t.Fatal("Expected trip to be returned")
	}
	if trip.Status != domain.StatusMatching {
		t.Errorf("Expected status %s, got %s", domain.StatusMatching, trip.Status)
	}
	if eventsPublished != 2 {
		t.Errorf("Expected 2 Kafka events, got %d", eventsPublished)
	}
}

func TestExecuteCreateTripSaga_OSRMFailure(t *testing.T) {
	orc, _, osrmClient, _, _ := setupTestOrchestrator()

	osrmClient.GetRouteFunc = func(ctx context.Context, origin, destination domain.Location) (*osrm.RouteResult, error) {
		return nil, errors.New("osrm down")
	}

	cmd := CreateTripCmd{}
	_, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error due to OSRM failure, got nil")
	}
}

func TestExecuteCreateTripSaga_DistanceExceeded(t *testing.T) {
	orc, _, osrmClient, _, _ := setupTestOrchestrator()

	osrmClient.GetRouteFunc = func(ctx context.Context, origin, destination domain.Location) (*osrm.RouteResult, error) {
		return &osrm.RouteResult{DistanceKm: 150.0, DurationSecs: 3600}, nil
	}

	cmd := CreateTripCmd{}
	_, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error due to max distance exceeded, got nil")
	}
}

func TestExecuteCreateTripSaga_PaymentFailure(t *testing.T) {
	orc, _, _, _, paymentClient := setupTestOrchestrator()

	paymentClient.AuthorizeHoldFunc = func(ctx context.Context, tripID, riderID string, amountCents int64, currency, paymentMethodID string) (string, error) {
		return "", errors.New("insufficient funds")
	}

	cmd := CreateTripCmd{}
	_, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error due to payment failure, got nil")
	}
}

func TestExecuteCreateTripSaga_DBFailure_TriggersCompensation(t *testing.T) {
	orc, repo, _, _, paymentClient := setupTestOrchestrator()

	repo.CreateFunc = func(ctx context.Context, trip *domain.Trip) error {
		return errors.New("db down")
	}

	cmd := CreateTripCmd{}
	_, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error due to DB failure, got nil")
	}
	if !paymentClient.ReleaseHoldCalled {
		t.Error("Expected ReleaseHold to be called as a compensation step")
	}
}

func TestExecuteCreateTripSaga_MatchmakingUpdateFailure_TriggersFullCompensation(t *testing.T) {
	orc, repo, _, _, paymentClient := setupTestOrchestrator()

	repo.UpdateStatusFunc = func(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
		if newStatus == domain.StatusMatching {
			return errors.New("db down during matchmaking update")
		}
		return nil
	}

	cmd := CreateTripCmd{}
	_, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error due to DB failure, got nil")
	}
	if !paymentClient.ReleaseHoldCalled {
		t.Error("Expected ReleaseHold to be called as a compensation step after Matchmaking update failed")
	}
}
