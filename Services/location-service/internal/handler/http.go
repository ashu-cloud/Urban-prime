package handler

import (
	"net/http"

	"github.com/cab-booking/pkg/httpserver"
	locationv1 "github.com/cab-booking/proto/gen/location/v1"
)

func (h *LocationHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpserver.Health("location-service"))
	mux.HandleFunc("GET /api/v1/health", httpserver.Health("location-service"))
	mux.HandleFunc("POST /api/v1/location/driver", h.UpdateDriverLocationHTTP)
	mux.HandleFunc("POST /location/driver", h.UpdateDriverLocationHTTP)
	mux.HandleFunc("GET /api/v1/location/driver/{id}", h.GetDriverLocationHTTP)
	mux.HandleFunc("GET /location/driver/{id}", h.GetDriverLocationHTTP)
	return mux
}

func (h *LocationHandler) UpdateDriverLocationHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DriverID  string  `json:"driverId"`
		DriverAlt string  `json:"driver_id"`
		TripID    string  `json:"tripId"`
		TripAlt   string  `json:"trip_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Heading   float32 `json:"heading"`
		Bearing   float32 `json:"bearing"`
		SpeedKmh  float32 `json:"speed_kmh"`
	}
	if err := httpserver.DecodeJSON(r, &body); err != nil {
		httpserver.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	bearing := body.Bearing
	if bearing == 0 {
		bearing = body.Heading
	}
	resp, err := h.UpdateDriverLocation(r.Context(), &locationv1.UpdateDriverLocationRequest{
		DriverId:  httpserver.FirstNonEmpty(body.DriverID, body.DriverAlt),
		TripId:    httpserver.FirstNonEmpty(body.TripID, body.TripAlt),
		Latitude:  body.Latitude,
		Longitude: body.Longitude,
		Bearing:   bearing,
		SpeedKmh:  body.SpeedKmh,
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, resp)
}

func (h *LocationHandler) GetDriverLocationHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.GetDriverLocation(r.Context(), &locationv1.GetDriverLocationRequest{
		DriverId: r.PathValue("id"),
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, resp)
}
