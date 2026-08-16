package tripv1

type TripStatus int32

const (
	TripStatus_TRIP_STATUS_UNSPECIFIED    TripStatus = 0
	TripStatus_TRIP_STATUS_REQUESTED      TripStatus = 1
	TripStatus_TRIP_STATUS_MATCHING       TripStatus = 2
	TripStatus_TRIP_STATUS_ASSIGNED       TripStatus = 3
	TripStatus_TRIP_STATUS_IN_PROGRESS    TripStatus = 4
	TripStatus_TRIP_STATUS_COMPLETED      TripStatus = 5
	TripStatus_TRIP_STATUS_CANCELLED      TripStatus = 6
	TripStatus_TRIP_STATUS_PAYMENT_FAILED TripStatus = 7
)

func (x TripStatus) String() string {
	switch x {
	case TripStatus_TRIP_STATUS_UNSPECIFIED:
		return "TRIP_STATUS_UNSPECIFIED"
	case TripStatus_TRIP_STATUS_REQUESTED:
		return "TRIP_STATUS_REQUESTED"
	case TripStatus_TRIP_STATUS_MATCHING:
		return "TRIP_STATUS_MATCHING"
	case TripStatus_TRIP_STATUS_ASSIGNED:
		return "TRIP_STATUS_ASSIGNED"
	case TripStatus_TRIP_STATUS_IN_PROGRESS:
		return "TRIP_STATUS_IN_PROGRESS"
	case TripStatus_TRIP_STATUS_COMPLETED:
		return "TRIP_STATUS_COMPLETED"
	case TripStatus_TRIP_STATUS_CANCELLED:
		return "TRIP_STATUS_CANCELLED"
	case TripStatus_TRIP_STATUS_PAYMENT_FAILED:
		return "TRIP_STATUS_PAYMENT_FAILED"
	default:
		return ""
	}
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

func (x *Location) GetLatitude() float64 {
	if x != nil {
		return x.Latitude
	}
	return 0
}

func (x *Location) GetLongitude() float64 {
	if x != nil {
		return x.Longitude
	}
	return 0
}

func (x *Location) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

type Money struct {
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

func (x *Money) GetCurrency() string {
	if x != nil {
		return x.Currency
	}
	return ""
}

func (x *Money) GetAmountCents() int64 {
	if x != nil {
		return x.AmountCents
	}
	return 0
}

type Trip struct {
	TripId                   string     `json:"trip_id"`
	RiderId                  string     `json:"rider_id"`
	DriverId                 string     `json:"driver_id"`
	PickupLocation           *Location  `json:"pickup_location"`
	DropoffLocation          *Location  `json:"dropoff_location"`
	Status                   TripStatus `json:"status"`
	EstimatedFare            *Money     `json:"estimated_fare"`
	FinalFare                *Money     `json:"final_fare"`
	DistanceKm               float64    `json:"distance_km"`
	EstimatedDurationSeconds int64      `json:"estimated_duration_seconds"`
	CreatedAt                int64      `json:"created_at"`
	UpdatedAt                int64      `json:"updated_at"`
}

func (x *Trip) GetTripId() string {
	if x != nil {
		return x.TripId
	}
	return ""
}

func (x *Trip) GetRiderId() string {
	if x != nil {
		return x.RiderId
	}
	return ""
}

func (x *Trip) GetDriverId() string {
	if x != nil {
		return x.DriverId
	}
	return ""
}

func (x *Trip) GetPickupLocation() *Location {
	if x != nil {
		return x.PickupLocation
	}
	return nil
}

func (x *Trip) GetDropoffLocation() *Location {
	if x != nil {
		return x.DropoffLocation
	}
	return nil
}

func (x *Trip) GetStatus() TripStatus {
	if x != nil {
		return x.Status
	}
	return TripStatus_TRIP_STATUS_UNSPECIFIED
}

func (x *Trip) GetEstimatedFare() *Money {
	if x != nil {
		return x.EstimatedFare
	}
	return nil
}

func (x *Trip) GetDistanceKm() float64 {
	if x != nil {
		return x.DistanceKm
	}
	return 0
}

func (x *Trip) GetEstimatedDurationSeconds() int64 {
	if x != nil {
		return x.EstimatedDurationSeconds
	}
	return 0
}

type CreateTripRequest struct {
	RiderId         string    `json:"rider_id"`
	PickupLocation  *Location `json:"pickup_location"`
	DropoffLocation *Location `json:"dropoff_location"`
	VehicleType     string    `json:"vehicle_type"`
	PaymentMethodId string    `json:"payment_method_id"`
}

func (x *CreateTripRequest) GetRiderId() string {
	if x != nil {
		return x.RiderId
	}
	return ""
}

func (x *CreateTripRequest) GetPickupLocation() *Location {
	if x != nil {
		return x.PickupLocation
	}
	return nil
}

func (x *CreateTripRequest) GetDropoffLocation() *Location {
	if x != nil {
		return x.DropoffLocation
	}
	return nil
}

func (x *CreateTripRequest) GetVehicleType() string {
	if x != nil {
		return x.VehicleType
	}
	return ""
}

func (x *CreateTripRequest) GetPaymentMethodId() string {
	if x != nil {
		return x.PaymentMethodId
	}
	return ""
}

type CreateTripResponse struct {
	Trip *Trip `json:"trip"`
}

func (x *CreateTripResponse) GetTrip() *Trip {
	if x != nil {
		return x.Trip
	}
	return nil
}

type GetTripRequest struct {
	TripId string `json:"trip_id"`
}

func (x *GetTripRequest) GetTripId() string {
	if x != nil {
		return x.TripId
	}
	return ""
}

type GetTripResponse struct {
	Trip *Trip `json:"trip"`
}

func (x *GetTripResponse) GetTrip() *Trip {
	if x != nil {
		return x.Trip
	}
	return nil
}

type CancelTripRequest struct {
	TripId string `json:"trip_id"`
	Reason string `json:"reason"`
}

func (x *CancelTripRequest) GetTripId() string {
	if x != nil {
		return x.TripId
	}
	return ""
}

func (x *CancelTripRequest) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

type CancelTripResponse struct {
	Success bool  `json:"success"`
	Trip    *Trip `json:"trip"`
}

func (x *CancelTripResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *CancelTripResponse) GetTrip() *Trip {
	if x != nil {
		return x.Trip
	}
	return nil
}
