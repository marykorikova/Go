package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/repository"
	"github.com/marykorikova/Go/internal/service"
)

type Handler struct {
	svc *service.TripService
}

func NewHandler(svc *service.TripService) *Handler {
	return &Handler{svc: svc}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code string, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":     "https://tripgo.example/problems/" + code,
		"title":    code,
		"status":   status,
		"detail":   detail,
		"instance": r.URL.Path,
		"code":     code,
	})
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params apiGen.CreateTripParams) {
	var req apiGen.Trip
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.Price < 0 || req.UserId == uuid.Nil || req.DriverId == uuid.Nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid fields")
		return
	}

	trip, err := h.svc.CreateTrip(r.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			writeError(w, r, http.StatusConflict, "driver_busy", "Driver already has an active trip")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+trip.Id.String())
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trip)
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	trip, err := h.svc.GetTrip(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trip)
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	trip, err := h.svc.FinishTrip(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}
		if errors.Is(err, repository.ErrTripFinished) {
			writeError(w, r, http.StatusConflict, "trip_completed", "Trip is already completed")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trip)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// В реальности тут нужно проверять Ping БД, но для простоты пока 200
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}
func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}
