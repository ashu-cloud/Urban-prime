package driverv1

type DriverStatus int32

const (
	DriverStatus_DRIVER_STATUS_UNSPECIFIED DriverStatus = 0
	DriverStatus_DRIVER_STATUS_OFFLINE     DriverStatus = 1
	DriverStatus_DRIVER_STATUS_AVAILABLE   DriverStatus = 2
	DriverStatus_DRIVER_STATUS_ON_TRIP     DriverStatus = 3
)

func (x DriverStatus) String() string {
	switch x {
	case DriverStatus_DRIVER_STATUS_UNSPECIFIED:
		return "DRIVER_STATUS_UNSPECIFIED"
	case DriverStatus_DRIVER_STATUS_OFFLINE:
		return "DRIVER_STATUS_OFFLINE"
	case DriverStatus_DRIVER_STATUS_AVAILABLE:
		return "DRIVER_STATUS_AVAILABLE"
	case DriverStatus_DRIVER_STATUS_ON_TRIP:
		return "DRIVER_STATUS_ON_TRIP"
	default:
		return ""
	}
}

type VehicleType int32

const (
	VehicleType_VEHICLE_TYPE_UNSPECIFIED VehicleType = 0
	VehicleType_VEHICLE_TYPE_SEDAN       VehicleType = 1
	VehicleType_VEHICLE_TYPE_SUV         VehicleType = 2
	VehicleType_VEHICLE_TYPE_PREMIUM     VehicleType = 3
	VehicleType_VEHICLE_TYPE_BIKE        VehicleType = 4
)

func (x VehicleType) String() string {
	switch x {
	case VehicleType_VEHICLE_TYPE_UNSPECIFIED:
		return "VEHICLE_TYPE_UNSPECIFIED"
	case VehicleType_VEHICLE_TYPE_SEDAN:
		return "VEHICLE_TYPE_SEDAN"
	case VehicleType_VEHICLE_TYPE_SUV:
		return "VEHICLE_TYPE_SUV"
	case VehicleType_VEHICLE_TYPE_PREMIUM:
		return "VEHICLE_TYPE_PREMIUM"
	case VehicleType_VEHICLE_TYPE_BIKE:
		return "VEHICLE_TYPE_BIKE"
	default:
		return ""
	}
}

type Driver struct {
	DriverId     string       `json:"driver_id"`
	Name         string       `json:"name"`
	Phone        string       `json:"phone"`
	Email        string       `json:"email"`
	Status       DriverStatus `json:"status"`
	VehicleType  VehicleType  `json:"vehicle_type"`
	VehiclePlate string       `json:"vehicle_plate"`
	VehicleModel string       `json:"vehicle_model"`
	Rating       float64      `json:"rating"`
	TotalTrips   int32        `json:"total_trips"`
	CreatedAt    int64        `json:"created_at"`
	UpdatedAt    int64        `json:"updated_at"`
}

func (x *Driver) GetDriverId() string {
	if x != nil {
		return x.DriverId
	}
	return ""
}

func (x *Driver) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Driver) GetPhone() string {
	if x != nil {
		return x.Phone
	}
	return ""
}

func (x *Driver) GetStatus() DriverStatus {
	if x != nil {
		return x.Status
	}
	return DriverStatus_DRIVER_STATUS_UNSPECIFIED
}

type MatchDriverRequest struct {
	TripId          string  `json:"trip_id"`
	PickupLatitude  float64 `json:"pickup_latitude"`
	PickupLongitude float64 `json:"pickup_longitude"`
	VehicleType     string  `json:"vehicle_type"`
}

type MatchDriverResponse struct {
	Matched  bool    `json:"matched"`
	DriverId string  `json:"driver_id"`
	Driver   *Driver `json:"driver"`
}

type UpdateDriverStatusRequest struct {
	DriverId  string       `json:"driver_id"`
	Status    DriverStatus `json:"status"`
	Latitude  float64      `json:"latitude"`
	Longitude float64      `json:"longitude"`
}

type UpdateDriverStatusResponse struct {
	Success bool    `json:"success"`
	Driver  *Driver `json:"driver"`
}

type RegisterDriverRequest struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	VehicleType  string `json:"vehicle_type"`
	VehiclePlate string `json:"vehicle_plate"`
	VehicleModel string `json:"vehicle_model"`
}

type RegisterDriverResponse struct {
	Driver *Driver `json:"driver"`
}

type GetDriverRequest struct {
	DriverId string `json:"driver_id"`
}

type GetDriverResponse struct {
	Driver *Driver `json:"driver"`
}
