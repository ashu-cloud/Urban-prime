package saga

import (
	"context"
	"sync"
	"testing"

	"github.com/cab-booking/trip-service/internal/domain"
)

func TestBookingFlowE2E_AssignThenCompensateUnknown(t *testing.T) {
	orc, _, _, _, payment := setupTestOrchestrator()

	cmd := CreateTripCmd{
		RiderID:         "rider-e2e",
		Pickup:          domain.Location{Latitude: 12.97, Longitude: 77.59, Address: "Pickup"},
		Dropoff:         domain.Location{Latitude: 12.94, Longitude: 77.62, Address: "Dropoff"},
		VehicleType:     "SEDAN",
		PaymentMethodID: "pm_card",
	}

	trip, err := orc.ExecuteCreateTripSaga(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if trip.Status != domain.StatusMatching {
		t.Fatalf("status %s", trip.Status)
	}
	if trip.FinalFareCents <= 0 {
		t.Fatal("fare must be computed")
	}

	if err := orc.AssignDriverToTrip(context.Background(), trip.ID, "drv-e2e"); err != nil {
		t.Fatalf("assign: %v", err)
	}

	payment.ReleaseHoldCalled = false
	orc.CompensateTripCreation(context.Background(), trip.ID, "rider cancelled")
	if !payment.ReleaseHoldCalled {
		t.Fatal("cancel must release payment hold")
	}
}

func TestConcurrentSagaCreates(t *testing.T) {
	orc, _, _, _, _ := setupTestOrchestrator()
	const n = 40
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := orc.ExecuteCreateTripSaga(context.Background(), CreateTripCmd{
				RiderID: "rider",
				Pickup:  domain.Location{Latitude: 12.97, Longitude: 77.59},
				Dropoff: domain.Location{Latitude: 12.94, Longitude: 77.62},
			})
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
