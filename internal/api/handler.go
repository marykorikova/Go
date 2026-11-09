package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	apiGen "github.com/marykorikova/Go/internal/generated"
	tripcreate "github.com/marykorikova/Go/internal/http/trip/create"
	tripfinish "github.com/marykorikova/Go/internal/http/trip/finish"
	tripget "github.com/marykorikova/Go/internal/http/trip/get"
)

type Handler struct {
	pool          *pgxpool.Pool
	queryTimeout  time.Duration
	createHandler *tripcreate.Handler
	getHandler    *tripget.Handler
	finishHandler *tripfinish.Handler
}

func NewHandlerWithUsecases(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
	createHandler *tripcreate.Handler,
	getHandler *tripget.Handler,
	finishHandler *tripfinish.Handler,
) *Handler {
	return &Handler{
		pool:          pool,
		queryTimeout:  queryTimeout,
		createHandler: createHandler,
		getHandler:    getHandler,
		finishHandler: finishHandler,
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
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	h.createHandler.Handle(w, r.WithContext(ctx))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	h.getHandler.Handle(w, r.WithContext(ctx), tripId)
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId apiGen.TripId) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	h.finishHandler.Handle(w, r.WithContext(ctx), tripId)
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
