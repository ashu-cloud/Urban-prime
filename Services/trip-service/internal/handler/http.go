package handler

import (
	"net/http"

	"github.com/cab-booking/pkg/httpserver"
	tripv1 "github.com/cab-booking/proto/gen/trip/v1"
)

type createTripHTTPRequest struct {
	RiderID         string  `json:"riderId"`
	RiderIDSnake    string  `json:"rider_id"`
	PickupAddress   string  `json:"pickupAddress"`
	PickupLat       float64 `json:"pickupLat"`
	PickupLng       float64 `json:"pickupLng"`
	DropoffAddress  string  `json:"dropoffAddress"`
	DropoffLat      float64 `json:"dropoffLat"`
	DropoffLng      float64 `json:"dropoffLng"`
	VehicleType     string  `json:"vehicleType"`
	VehicleTypeAlt  string  `json:"vehicle_type"`
	PaymentMethodID string  `json:"paymentMethodId"`
	PaymentAlt      string  `json:"payment_method_id"`
	PickupLocation  *tripv1.Location `json:"pickup_location"`
	DropoffLocation *tripv1.Location `json:"dropoff_location"`
}

func (h *TripHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", httpserver.Health("trip-service"))
	mux.HandleFunc("GET /api/v1/health", httpserver.Health("trip-service"))
	mux.HandleFunc("POST /api/v1/trips", h.CreateTripHTTP)
	mux.HandleFunc("POST /trips", h.CreateTripHTTP)
	mux.HandleFunc("GET /api/v1/trips/{id}", h.GetTripHTTP)
	mux.HandleFunc("GET /trips/{id}", h.GetTripHTTP)
	mux.HandleFunc("POST /api/v1/trips/{id}/cancel", h.CancelTripHTTP)
	mux.HandleFunc("POST /trips/{id}/cancel", h.CancelTripHTTP)
	return mux
}

func (h *TripHandler) CreateTripHTTP(w http.ResponseWriter, r *http.Request) {
	var body createTripHTTPRequest
	if err := httpserver.DecodeJSON(r, &body); err != nil {
		httpserver.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	pickup := body.PickupLocation
	if pickup == nil {
		pickup = &tripv1.Location{
			Latitude:  body.PickupLat,
			Longitude: body.PickupLng,
			Address:   body.PickupAddress,
		}
	}
	dropoff := body.DropoffLocation
	if dropoff == nil {
		dropoff = &tripv1.Location{
			Latitude:  body.DropoffLat,
			Longitude: body.DropoffLng,
			Address:   body.DropoffAddress,
		}
	}

	resp, err := h.CreateTrip(r.Context(), &tripv1.CreateTripRequest{
		RiderId:         httpserver.FirstNonEmpty(body.RiderID, body.RiderIDSnake),
		PickupLocation:  pickup,
		DropoffLocation: dropoff,
		VehicleType:     httpserver.FirstNonEmpty(body.VehicleType, body.VehicleTypeAlt),
		PaymentMethodId: httpserver.FirstNonEmpty(body.PaymentMethodID, body.PaymentAlt),
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, mapTripHTTP(resp.Trip))
}

func (h *TripHandler) GetTripHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp, err := h.GetTrip(r.Context(), &tripv1.GetTripRequest{TripId: id})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, mapTripHTTP(resp.Trip))
}

func (h *TripHandler) CancelTripHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	_ = httpserver.DecodeJSON(r, &body)
	resp, err := h.CancelTrip(r.Context(), &tripv1.CancelTripRequest{
		TripId: r.PathValue("id"),
		Reason: body.Reason,
	})
	if err != nil {
		httpserver.WriteGRPCError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"success": resp.Success,
		"trip":    mapTripHTTP(resp.Trip),
	})
}

func mapTripHTTP(t *tripv1.Trip) map[string]any {
	if t == nil {
		return nil
	}
	out := map[string]any{
		"tripId":   t.TripId,
		"riderId":  t.RiderId,
		"driverId": t.DriverId,
		"status":   t.Status.String(),
		"distanceKm": t.DistanceKm,
		"estimatedMinutes": t.EstimatedDurationSeconds / 60,
		"createdAt": t.CreatedAt,
	}
	if t.EstimatedFare != nil {
		out["fare"] = float64(t.EstimatedFare.AmountCents) / 100.0
		out["fareAmount"] = t.EstimatedFare.AmountCents
		out["currency"] = t.EstimatedFare.Currency
	}
	if t.PickupLocation != nil {
		out["pickupLocation"] = t.PickupLocation
	}
	if t.DropoffLocation != nil {
		out["dropoffLocation"] = t.DropoffLocation
	}
	return out
}
