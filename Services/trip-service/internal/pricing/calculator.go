package pricing

import (
	"math"

	"github.com/cab-booking/trip-service/internal/config"
)

// Calculator handles Uber-model fare pricing calculations.
type Calculator struct {
	baseFareCents    int64   // Base booking fee in cents (e.g. 3000 = ₹30.00)
	perKmRateCents   int64   // Charge per km in cents (e.g. 1500 = ₹15.00/km)
	perMinRateCents  int64   // Charge per minute in cents (e.g. 100 = ₹1.00/min)
	defaultSurgeMult float64 // Default surge multiplier (1.0 = normal pricing)
}

// NewCalculator initializes the pricing calculator with rates from configuration
func NewCalculator(cfg *config.Config) *Calculator {
	return &Calculator{
		baseFareCents:    cfg.BaseFareCents,
		perKmRateCents:   cfg.PerKmRateCents,
		perMinRateCents:  cfg.PerMinRateCents,
		defaultSurgeMult: cfg.DefaultSurgeMult,
	}
}

// FareBreakdown holds detailed breakdown of components in final fare calculation
type FareBreakdown struct {
	BaseFareCents   int64   `json:"base_fare_cents"`
	DistanceFare    int64   `json:"distance_fare_cents"`
	DurationFare    int64   `json:"duration_fare_cents"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	TotalFareCents  int64   `json:"total_fare_cents"`
}

// CalculateFare computes trip price using standard Uber fare model formula:
//
//	Total = (BaseFare + (DistanceKm * PerKmRate) + (DurationMins * PerMinRate)) * SurgeMultiplier
//
// WHY CENTS INSTEAD OF FLOATS FOR MONEY?
// Floating-point numbers (float64/float32) in computers cause rounding errors (e.g., 0.1 + 0.2 = 0.30000000000000004).
// In financial systems and ride-sharing platforms, prices are always calculated in integer subunits (cents/paise)
// to eliminate floating point precision bugs completely!
func (c *Calculator) CalculateFare(distanceKm float64, durationSecs int64, surgeMultiplier float64) FareBreakdown {
	// Fallback to default multiplier if invalid surge parameter provided
	if surgeMultiplier <= 0 {
		surgeMultiplier = c.defaultSurgeMult
	}
	if distanceKm < 0 {
		distanceKm = 0
	}
	if durationSecs < 0 {
		durationSecs = 0
	}

	// Convert travel duration seconds into minutes (rounding up using math.Ceil)
	durationMins := math.Ceil(float64(durationSecs) / 60.0)
	if durationMins < 1 {
		durationMins = 1 // Minimum 1 minute charge
	}

	// Calculate distance component
	distanceFare := int64(math.Round(distanceKm * float64(c.perKmRateCents)))

	// Calculate duration component
	durationFare := int64(durationMins * float64(c.perMinRateCents))

	// Subtotal sum before surge multiplier application
	subtotal := c.baseFareCents + distanceFare + durationFare

	// Apply surge multiplier (e.g. 1.5x during rush hour)
	total := int64(math.Round(float64(subtotal) * surgeMultiplier))

	return FareBreakdown{
		BaseFareCents:   c.baseFareCents,
		DistanceFare:    distanceFare,
		DurationFare:    durationFare,
		SurgeMultiplier: surgeMultiplier,
		TotalFareCents:  total,
	}
}
