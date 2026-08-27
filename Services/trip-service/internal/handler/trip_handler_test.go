package handler

import (
	"context"
	"errors"
	"sync"
	"testing"

	tripv1 "github.com/cab-booking/proto/gen/trip/v1"
	"github.com/cab-booking/trip-service/internal/config"
	"github.com/cab-booking/trip-service/internal/domain"
	"github.com/cab-booking/trip-service/internal/kafka"
	"github.com/cab-booking/trip-service/internal/osrm"
	"github.com/cab-booking/trip-service/internal/pricing"
	"github.com/cab-booking/trip-service/internal/saga"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memTripRepo struct {
	mu    sync.Mutex
	trips map[string]*domain.Trip
}

func newMemTripRepo() *memTripRepo {
	return &memTripRepo{trips: make(map[string]*domain.Trip)}
}

func (m *memTripRepo) Create(ctx context.Context, trip *domain.Trip) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *trip
	m.trips[trip.ID] = &cp
	return nil
}

func (m *memTripRepo) UpdateStatus(ctx context.Context, id string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	trip, ok := m.trips[id]
	if !ok {
		return errors.New("not found")
	}
	trip.Status = newStatus
	trip.SagaLog = append(trip.SagaLog, step)
	return nil
}

func (m *memTripRepo) AssignDriver(ctx context.Context, tripID, driverID string, newStatus domain.TripStatus, step domain.SagaStepLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	trip, ok := m.trips[tripID]
	if !ok {
		return errors.New("not found")
	}
	trip.DriverID = &driverID
	trip.Status = newStatus
	return nil
}

func (m *memTripRepo) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	trip, ok := m.trips[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *trip
	return &cp, nil
}

type stubOSRM struct{}

func (s stubOSRM) GetRoute(ctx context.Context, origin, destination domain.Location) (*osrm.RouteResult, error) {
	return &osrm.RouteResult{DistanceKm: 8.0, DurationSecs: 720}, nil
}

type stubKafka struct{}

func (s stubKafka) PublishTripEvent(ctx context.Context, topic string, payload kafka.TripEventPayload) error {
	return nil
}

type stubPay struct{}

func (s stubPay) AuthorizeHold(ctx context.Context, tripID, riderID string, amountCents int64, currency, paymentMethodID string) (string, error) {
	return "txn_ok", nil
}
func (s stubPay) ReleaseHold(ctx context.Context, transactionID, tripID, reason string) error {
	return nil
}

func newTripHandler() (*TripHandler, *memTripRepo) {
	repo := newMemTripRepo()
	calc := pricing.NewCalculator(&config.Config{BaseFareCents: 3000, PerKmRateCents: 1500, PerMinRateCents: 100, DefaultSurgeMult: 1})
	orc := saga.NewOrchestrator(repo, stubOSRM{}, calc, stubKafka{}, stubPay{})
	return NewTripHandler(orc, repo), repo
}

func validCreateReq() *tripv1.CreateTripRequest {
	return &tripv1.CreateTripRequest{
		RiderId: "rider-1",
		PickupLocation: &tripv1.Location{
			Latitude: 12.9716, Longitude: 77.5946, Address: "MG Road",
		},
		DropoffLocation: &tripv1.Location{
			Latitude: 12.9352, Longitude: 77.6245, Address: "Koramangala",
		},
		VehicleType:     "SEDAN",
		PaymentMethodId: "pm_card",
	}
}

func TestCreateTripSuccess(t *testing.T) {
	h, _ := newTripHandler()
	resp, err := h.CreateTrip(context.Background(), validCreateReq())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Trip.TripId == "" || resp.Trip.RiderId != "rider-1" {
		t.Fatalf("unexpected trip %+v", resp.Trip)
	}
	if resp.Trip.Status != tripv1.TripStatus_TRIP_STATUS_MATCHING {
		t.Fatalf("status %v", resp.Trip.Status)
	}
}

func TestCreateTripValidation(t *testing.T) {
	h, _ := newTripHandler()
	_, err := h.CreateTrip(context.Background(), &tripv1.CreateTripRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing rider code %v", status.Code(err))
	}

	req := validCreateReq()
	req.PickupLocation.Latitude = 999
	_, err = h.CreateTrip(context.Background(), req)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad coords code %v", status.Code(err))
	}
}

func TestGetAndCancelTrip(t *testing.T) {
	h, _ := newTripHandler()
	created, err := h.CreateTrip(context.Background(), validCreateReq())
	if err != nil {
		t.Fatal(err)
	}

	got, err := h.GetTrip(context.Background(), &tripv1.GetTripRequest{TripId: created.Trip.TripId})
	if err != nil {
		t.Fatal(err)
	}
	if got.Trip.TripId != created.Trip.TripId {
		t.Fatal("get mismatch")
	}

	cancelled, err := h.CancelTrip(context.Background(), &tripv1.CancelTripRequest{TripId: created.Trip.TripId, Reason: "changed mind"})
	if err != nil {
		t.Fatal(err)
	}
	if !cancelled.Success {
		t.Fatal("cancel failed")
	}

	_, err = h.GetTrip(context.Background(), &tripv1.GetTripRequest{TripId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing trip code %v", status.Code(err))
	}
}

func TestCancelCompletedTripRejected(t *testing.T) {
	h, repo := newTripHandler()
	created, _ := h.CreateTrip(context.Background(), validCreateReq())
	_ = repo.UpdateStatus(context.Background(), created.Trip.TripId, domain.StatusAssigned, domain.SagaStepLog{})
	_ = repo.UpdateStatus(context.Background(), created.Trip.TripId, domain.StatusInProgress, domain.SagaStepLog{})
	_ = repo.UpdateStatus(context.Background(), created.Trip.TripId, domain.StatusCompleted, domain.SagaStepLog{})

	_, err := h.CancelTrip(context.Background(), &tripv1.CancelTripRequest{TripId: created.Trip.TripId})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("cancel completed code %v", status.Code(err))
	}
}

func TestConcurrentCreateTrip(t *testing.T) {
	h, _ := newTripHandler()
	var wg sync.WaitGroup
	const n = 50
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := h.CreateTrip(context.Background(), validCreateReq())
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}
