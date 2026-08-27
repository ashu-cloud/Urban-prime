package dispatch

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cab-booking/driver-service/internal/domain"
	"github.com/cab-booking/driver-service/internal/kafka"
	"github.com/redis/go-redis/v9"
)

// Mocks

type MockGeoService struct {
	FindNearbyDriversFunc   func(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error)
	AcquireDispatchLockFunc func(ctx context.Context, driverID, tripID string, ttl time.Duration) (bool, error)
	ReleaseDispatchLockFunc func(ctx context.Context, driverID string) error
	RemoveDriverFunc        func(ctx context.Context, driverID string) error
	
	mu    sync.Mutex
	locks map[string]string // driverID -> tripID
}

func NewMockGeoService() *MockGeoService {
	return &MockGeoService{
		locks: make(map[string]string),
	}
}

func (m *MockGeoService) FindNearbyDrivers(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
	if m.FindNearbyDriversFunc != nil {
		return m.FindNearbyDriversFunc(ctx, lat, lng, radiusKm, limit)
	}
	return []redis.GeoLocation{
		{Name: "driver_1"},
		{Name: "driver_2"},
	}, nil
}

func (m *MockGeoService) AcquireDispatchLock(ctx context.Context, driverID, tripID string, ttl time.Duration) (bool, error) {
	if m.AcquireDispatchLockFunc != nil {
		return m.AcquireDispatchLockFunc(ctx, driverID, tripID, ttl)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.locks[driverID]; ok {
		return false, nil // Already locked
	}
	m.locks[driverID] = tripID
	return true, nil
}

func (m *MockGeoService) ReleaseDispatchLock(ctx context.Context, driverID string) error {
	if m.ReleaseDispatchLockFunc != nil {
		return m.ReleaseDispatchLockFunc(ctx, driverID)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.locks, driverID)
	return nil
}

func (m *MockGeoService) RemoveDriver(ctx context.Context, driverID string) error {
	if m.RemoveDriverFunc != nil {
		return m.RemoveDriverFunc(ctx, driverID)
	}
	return nil
}

type MockDriverRepo struct {
	GetByIDFunc      func(ctx context.Context, id string) (*domain.Driver, error)
	UpdateStatusFunc func(ctx context.Context, id string, status domain.DriverStatus) error

	mu      sync.Mutex
	status  map[string]domain.DriverStatus
}

func (m *MockDriverRepo) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.Lock()
	st := domain.StatusAvailable
	if m.status != nil {
		if s, ok := m.status[id]; ok {
			st = s
		}
	}
	m.mu.Unlock()
	return &domain.Driver{ID: id, Name: "Test Driver", Status: st}, nil
}
func (m *MockDriverRepo) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	m.mu.Lock()
	if m.status == nil {
		m.status = make(map[string]domain.DriverStatus)
	}
	m.status[id] = status
	m.mu.Unlock()
	return nil
}

type MockKafkaProducer struct {
	PublishMatchEventFunc func(ctx context.Context, topic string, payload kafka.MatchEventPayload) error
}

func (m *MockKafkaProducer) PublishMatchEvent(ctx context.Context, topic string, payload kafka.MatchEventPayload) error {
	if m.PublishMatchEventFunc != nil {
		return m.PublishMatchEventFunc(ctx, topic, payload)
	}
	return nil
}

func TestDispatchLoop_ConcurrencyRaceCondition(t *testing.T) {
	geoSvc := NewMockGeoService()
	repo := &MockDriverRepo{}
	prod := &MockKafkaProducer{}

	loop := NewDispatchLoop(geoSvc, repo, prod)

	// Simulate driver accepting immediately
	loop.SimulateDriverResponse = func(ctx context.Context, driverID string) bool {
		time.Sleep(50 * time.Millisecond) // Give the race condition a chance to overlap
		return true
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var driverTrip1, driverTrip2 *domain.Driver

	// Both trips look for drivers at the exact same location at the exact same time
	go func() {
		defer wg.Done()
		driverTrip1, _ = loop.FindAndDispatchDriver(context.Background(), "trip_1", 12.9, 77.5, "")
	}()

	go func() {
		defer wg.Done()
		driverTrip2, _ = loop.FindAndDispatchDriver(context.Background(), "trip_2", 12.9, 77.5, "")
	}()

	wg.Wait()

	if driverTrip1 == nil || driverTrip2 == nil {
		t.Fatalf("Expected both trips to get a driver, got trip1: %v, trip2: %v", driverTrip1, driverTrip2)
	}

	if driverTrip1.ID == driverTrip2.ID {
		t.Errorf("Race condition failed! Both trips were assigned the same driver: %s", driverTrip1.ID)
	}
}

func TestDispatchLoop_DriverDeclines(t *testing.T) {
	geoSvc := NewMockGeoService()
	repo := &MockDriverRepo{}
	prod := &MockKafkaProducer{}

	loop := NewDispatchLoop(geoSvc, repo, prod)

	// Simulate driver declining
	loop.SimulateDriverResponse = func(ctx context.Context, driverID string) bool {
		return false // decline
	}

	driver, _ := loop.FindAndDispatchDriver(context.Background(), "trip_3", 12.9, 77.5, "")

	if driver != nil {
		t.Fatalf("Expected no driver to accept, but got %s", driver.ID)
	}
}

func TestDispatchLoop_VehicleTypeMismatch(t *testing.T) {
	geoSvc := NewMockGeoService()
	repo := &MockDriverRepo{}
	prod := &MockKafkaProducer{}

	// Driver 1 is a SEDAN, Driver 2 is a SUV
	repo.GetByIDFunc = func(ctx context.Context, id string) (*domain.Driver, error) {
		if id == "driver_1" {
			return &domain.Driver{ID: id, Status: domain.StatusAvailable, VehicleType: "SEDAN"}, nil
		}
		if id == "driver_2" {
			return &domain.Driver{ID: id, Status: domain.StatusAvailable, VehicleType: "SUV"}, nil
		}
		return nil, nil
	}

	loop := NewDispatchLoop(geoSvc, repo, prod)
	loop.SimulateDriverResponse = func(ctx context.Context, driverID string) bool {
		return true // always accept
	}

	// Requesting an SUV should skip driver_1 and assign driver_2
	driver, _ := loop.FindAndDispatchDriver(context.Background(), "trip_suv", 12.9, 77.5, "SUV")

	if driver == nil {
		t.Fatal("Expected an SUV driver to be assigned")
	}
	if driver.ID != "driver_2" {
		t.Errorf("Expected driver_2 to be assigned, got %s", driver.ID)
	}
}

func TestDispatchLoop_NoCandidates(t *testing.T) {
	geoSvc := NewMockGeoService()
	geoSvc.FindNearbyDriversFunc = func(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
		return nil, nil
	}
	loop := NewDispatchLoop(geoSvc, &MockDriverRepo{}, &MockKafkaProducer{})
	driver, err := loop.FindAndDispatchDriver(context.Background(), "trip_empty", 12.9, 77.5, "SEDAN")
	if err != nil {
		t.Fatal(err)
	}
	if driver != nil {
		t.Fatal("expected no driver")
	}
}

func TestDispatchLoop_HighContention(t *testing.T) {
	geoSvc := NewMockGeoService()
	geoSvc.FindNearbyDriversFunc = func(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]redis.GeoLocation, error) {
		return []redis.GeoLocation{{Name: "driver_1"}, {Name: "driver_2"}, {Name: "driver_3"}}, nil
	}
	repo := &MockDriverRepo{}
	loop := NewDispatchLoop(geoSvc, repo, &MockKafkaProducer{})
	loop.SimulateDriverResponse = func(ctx context.Context, driverID string) bool { return true }

	const trips = 3
	var wg sync.WaitGroup
	wg.Add(trips)
	assigned := make(chan string, trips)
	for i := 0; i < trips; i++ {
		go func(i int) {
			defer wg.Done()
			d, _ := loop.FindAndDispatchDriver(context.Background(), fmt.Sprintf("trip_%d", i), 12.9, 77.5, "")
			if d != nil {
				assigned <- d.ID
			}
		}(i)
	}
	wg.Wait()
	close(assigned)

	seen := map[string]bool{}
	count := 0
	for id := range assigned {
		count++
		if seen[id] {
			t.Fatalf("driver %s assigned twice", id)
		}
		seen[id] = true
	}
	if count != trips {
		t.Fatalf("expected %d assignments, got %d", trips, count)
	}
}
