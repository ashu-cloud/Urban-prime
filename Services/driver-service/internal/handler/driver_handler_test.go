package handler

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/cab-booking/driver-service/internal/dispatch"
	"github.com/cab-booking/driver-service/internal/domain"
	"github.com/cab-booking/driver-service/internal/geo"
	"github.com/cab-booking/driver-service/internal/kafka"
	driverv1 "github.com/cab-booking/proto/gen/driver/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memDriverRepo struct {
	mu      sync.Mutex
	drivers map[string]*domain.Driver
}

func newMemDriverRepo() *memDriverRepo {
	return &memDriverRepo{drivers: make(map[string]*domain.Driver)}
}

func (m *memDriverRepo) CreateDriver(ctx context.Context, driver *domain.Driver) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *driver
	m.drivers[driver.ID] = &cp
	return nil
}

func (m *memDriverRepo) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.drivers[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *d
	return &cp, nil
}

func (m *memDriverRepo) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.drivers[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	d.Status = status
	return nil
}

type stubMatchProducer struct{}

func (s stubMatchProducer) PublishMatchEvent(ctx context.Context, topic string, payload kafka.MatchEventPayload) error {
	return nil
}

func newDriverHandler() *DriverHandler {
	repo := newMemDriverRepo()
	geoSvc := geo.NewGeoService(nil)
	loop := dispatch.NewDispatchLoop(geoSvc, repo, stubMatchProducer{})
	loop.SimulateDriverResponse = func(ctx context.Context, driverID string) bool { return true }
	return NewDriverHandler(loop, geoSvc, repo)
}

func TestRegisterAndGetDriver(t *testing.T) {
	h := newDriverHandler()
	resp, err := h.RegisterDriver(context.Background(), &driverv1.RegisterDriverRequest{
		Name:         "Priya",
		Phone:        "+15550001",
		Email:        "priya@example.com",
		VehiclePlate: "KA01AB1234",
		VehicleType:  "SEDAN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Driver.DriverId == "" || resp.Driver.Status != driverv1.DriverStatus_DRIVER_STATUS_OFFLINE {
		t.Fatalf("unexpected driver %+v", resp.Driver)
	}

	got, err := h.GetDriver(context.Background(), &driverv1.GetDriverRequest{DriverId: resp.Driver.DriverId})
	if err != nil {
		t.Fatal(err)
	}
	if got.Driver.Name != "Priya" {
		t.Fatalf("name %s", got.Driver.Name)
	}
}

func TestRegisterDriverValidation(t *testing.T) {
	h := newDriverHandler()
	_, err := h.RegisterDriver(context.Background(), &driverv1.RegisterDriverRequest{Name: "X"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code %v", status.Code(err))
	}
}

func TestUpdateDriverStatusEnforcesStateMachine(t *testing.T) {
	h := newDriverHandler()
	reg, _ := h.RegisterDriver(context.Background(), &driverv1.RegisterDriverRequest{
		Name: "A", Phone: "+1", VehiclePlate: "X1",
	})

	_, err := h.UpdateDriverStatus(context.Background(), &driverv1.UpdateDriverStatusRequest{
		DriverId:  reg.Driver.DriverId,
		Status:    driverv1.DriverStatus_DRIVER_STATUS_ON_TRIP,
		Latitude:  12.97,
		Longitude: 77.59,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("offline -> on_trip should fail, got %v", status.Code(err))
	}

	ok, err := h.UpdateDriverStatus(context.Background(), &driverv1.UpdateDriverStatusRequest{
		DriverId:  reg.Driver.DriverId,
		Status:    driverv1.DriverStatus_DRIVER_STATUS_AVAILABLE,
		Latitude:  12.97,
		Longitude: 77.59,
	})
	if err != nil || !ok.Success {
		t.Fatalf("go online failed: %v", err)
	}

	// Idempotent re-apply of AVAILABLE must succeed.
	_, err = h.UpdateDriverStatus(context.Background(), &driverv1.UpdateDriverStatusRequest{
		DriverId:  reg.Driver.DriverId,
		Status:    driverv1.DriverStatus_DRIVER_STATUS_AVAILABLE,
		Latitude:  12.98,
		Longitude: 77.60,
	})
	if err != nil {
		t.Fatalf("idempotent available failed: %v", err)
	}
}

func TestGetUnknownDriver(t *testing.T) {
	h := newDriverHandler()
	_, err := h.GetDriver(context.Background(), &driverv1.GetDriverRequest{DriverId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code %v", status.Code(err))
	}
}
