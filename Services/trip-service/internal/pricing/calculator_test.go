package pricing

import (
	"sync"
	"testing"

	"github.com/cab-booking/trip-service/internal/config"
)

func testCalc() *Calculator {
	return NewCalculator(&config.Config{
		BaseFareCents:    3000,
		PerKmRateCents:   1500,
		PerMinRateCents:  100,
		DefaultSurgeMult: 1.0,
	})
}

func TestCalculateFareStandardTrip(t *testing.T) {
	c := testCalc()
	fare := c.CalculateFare(10, 600, 1.0)
	// 3000 + 10*1500 + 10min*100 = 3000+15000+1000 = 19000
	if fare.TotalFareCents != 19000 {
		t.Fatalf("got %d want 19000", fare.TotalFareCents)
	}
	if fare.DistanceFare != 15000 || fare.DurationFare != 1000 {
		t.Errorf("breakdown %+v", fare)
	}
}

func TestCalculateFareSurge(t *testing.T) {
	c := testCalc()
	fare := c.CalculateFare(10, 600, 1.5)
	if fare.TotalFareCents != 28500 {
		t.Fatalf("got %d want 28500", fare.TotalFareCents)
	}
}

func TestCalculateFareInvalidSurgeFallsBack(t *testing.T) {
	c := testCalc()
	fare := c.CalculateFare(1, 60, 0)
	if fare.SurgeMultiplier != 1.0 {
		t.Fatalf("surge %v", fare.SurgeMultiplier)
	}
}

func TestCalculateFareMinimumOneMinute(t *testing.T) {
	c := testCalc()
	fare := c.CalculateFare(0, 0, 1.0)
	if fare.DurationFare != 100 {
		t.Fatalf("duration fare %d", fare.DurationFare)
	}
}

func TestCalculateFareNeverNegative(t *testing.T) {
	c := testCalc()
	fare := c.CalculateFare(-5, -10, 1.0)
	if fare.TotalFareCents < 0 {
		t.Fatalf("negative fare %d", fare.TotalFareCents)
	}
}

func TestCalculateFareConcurrent(t *testing.T) {
	c := testCalc()
	var wg sync.WaitGroup
	wg.Add(200)
	for i := 0; i < 200; i++ {
		go func() {
			defer wg.Done()
			_ = c.CalculateFare(8.4, 420, 1.2)
		}()
	}
	wg.Wait()
}
