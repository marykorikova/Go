package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/repository"
	"github.com/marykorikova/Go/internal/service"
)

type Handler struct {
	svc          *service.TripService
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewHandler(svc *service.TripService, pool *pgxpool.Pool, queryTimeout time.Duration) *Handler {
	return &Handler{
		svc:          svc,
		pool:         pool,
		queryTimeout: queryTimeout,
	}
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
	var req apiGen.TripData

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body or unknown fields")
		return
	}

	if req.Price <= 0 || req.UserId == uuid.Nil || req.DriverId == uuid.Nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid fields: price, user_id, driver_id are required")
		return
	}

	if req.StartPoint.Latitude < -90 || req.StartPoint.Latitude > 90 ||
		req.EndPoint.Latitude < -90 || req.EndPoint.Latitude > 90 ||
		req.StartPoint.Longitude < -180 || req.StartPoint.Longitude > 180 ||
		req.EndPoint.Longitude < -180 || req.EndPoint.Longitude > 180 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Coordinates out of range")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	newTrip := apiGen.Trip{
		Id:         uuid.New(),
		UserId:     req.UserId,
		DriverId:   req.DriverId,
		StartPoint: req.StartPoint,
		EndPoint:   req.EndPoint,
		Price:      req.Price,
		Status:     apiGen.Active, 
		StartedAt:  time.Now(),
	}

	trip, err := h.svc.CreateTrip(ctx, newTrip)
	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			writeError(w, r, http.StatusConflict, "driver_busy", "Driver already has an active trip")
			return
		}
		slog.Error("failed to create trip", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+trip.Id.String())
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trip)
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.svc.GetTrip(ctx, tripId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}
		slog.Error("failed to get trip", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trip)
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.svc.FinishTrip(ctx, tripId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}
		if errors.Is(err, repository.ErrTripFinished) {
			writeError(w, r, http.StatusConflict, "trip_completed", "Trip is already completed")
			return
		}
		slog.Error("failed to finish trip", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
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
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")
	if err := h.pool.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}
func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}