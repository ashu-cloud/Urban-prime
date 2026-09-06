package handler

import (
	"net/http"

	"github.com/cab-booking/pkg/httpserver"
	driverv1 "github.com/cab-booking/proto/gen/driver/v1"
)

func (h *DriverHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpserver.Health("driver-service"))
	mux.HandleFunc("GET /api/v1/health", httpserver.Health("driver-service"))
	mux.HandleFunc("POST /api/v1/drivers", h.RegisterDriverHTTP)
	mux.HandleFunc("POST /drivers", h.RegisterDriverHTTP)
	mux.HandleFunc("GET /api/v1/drivers/{id}", h.GetDriverHTTP)
	mux.HandleFunc("GET /drivers/{id}", h.GetDriverHTTP)
	mux.HandleFunc("PUT /api/v1/drivers/{id}/status", h.UpdateDriverStatusHTTP)
	mux.HandleFunc("POST /api/v1/drivers/{id}/status", h.UpdateDriverStatusHTTP)
	mux.HandleFunc("POST /api/v1/dispatch/{id}/respond", h.RespondDispatchHTTP)
	mux.HandleFunc("POST /api/v1/drivers/{id}/dispatch-response", h.RespondDispatchHTTP)
	mux.HandleFunc("POST /api/v1/drivers/dispatch/respond", h.RespondDispatchHTTP)
	return mux
}

func (h *DriverHandler) RegisterDriverHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		Phone        string `json:"phone"`
		Email        string `json:"email"`
		VehicleType  string `json:"vehicle_type"`
		VehicleType2 string `json:"vehicleType"`
		VehiclePlate string `json:"vehicle_plate"`
		VehiclePlate2 string `json:"vehiclePlate"`
		VehicleModel string `json:"vehicle_model"`
		VehicleModel2 string `json:"vehicleModel"`
	}
	if err := httpserver.DecodeJSON(r, &body); err != nil {
		httpserver.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.RegisterDriver(r.Context(), &driverv1.RegisterDriverRequest{
		Name:         body.Name,
		Phone:        body.Phone,
		Email:        body.Email,
		VehicleType:  httpserver.FirstNonEmpty(body.VehicleType, body.VehicleType2),
		VehiclePlate: httpserver.FirstNonEmpty(body.VehiclePlate, body.VehiclePlate2),
		VehicleModel: httpserver.FirstNonEmpty(body.VehicleModel, body.VehicleModel2),
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, resp.Driver)
}

func (h *DriverHandler) GetDriverHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.GetDriver(r.Context(), &driverv1.GetDriverRequest{DriverId: r.PathValue("id")})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, resp.Driver)
}

func (h *DriverHandler) UpdateDriverStatusHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status    string  `json:"status"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := httpserver.DecodeJSON(r, &body); err != nil {
		httpserver.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	statusEnum := driverv1.DriverStatus_DRIVER_STATUS_OFFLINE
	switch body.Status {
	case "AVAILABLE", "DRIVER_STATUS_AVAILABLE":
		statusEnum = driverv1.DriverStatus_DRIVER_STATUS_AVAILABLE
	case "ON_TRIP", "DRIVER_STATUS_ON_TRIP":
		statusEnum = driverv1.DriverStatus_DRIVER_STATUS_ON_TRIP
	}
	resp, err := h.UpdateDriverStatus(r.Context(), &driverv1.UpdateDriverStatusRequest{
		DriverId:  r.PathValue("id"),
		Status:    statusEnum,
		Latitude:  body.Latitude,
		Longitude: body.Longitude,
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, resp)
}

func (h *DriverHandler) RespondDispatchHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DriverID  string `json:"driver_id"`
		DriverID2 string `json:"driverId"`
		TripID    string `json:"trip_id"`
		TripID2   string `json:"tripId"`
		Accepted  *bool  `json:"accepted"`
		Accept    *bool  `json:"accept"`
	}
	if err := httpserver.DecodeJSON(r, &body); err != nil {
		httpserver.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	driverID := httpserver.FirstNonEmpty(r.PathValue("id"), body.DriverID, body.DriverID2)
	tripID := httpserver.FirstNonEmpty(body.TripID, body.TripID2)

	isAccepted := false
	if body.Accepted != nil {
		isAccepted = *body.Accepted
	} else if body.Accept != nil {
		isAccepted = *body.Accept
	}

	if driverID == "" {
		httpserver.WriteError(w, http.StatusBadRequest, "driver_id is required")
		return
	}

	if err := h.RespondToDispatch(r.Context(), driverID, tripID, isAccepted); err != nil {
		httpserver.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"driver_id": driverID,
		"trip_id":   tripID,
		"accepted":  isAccepted,
	})
}

