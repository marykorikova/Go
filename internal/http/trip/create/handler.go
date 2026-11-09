package create

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/domainerror"
	triphttp "github.com/marykorikova/Go/internal/http/trip"
	"github.com/marykorikova/Go/internal/model"
)

type tripCreator interface {
	Execute(ctx context.Context, input model.CreateTripInput) (model.Trip, error)
}

type Handler struct {
	usecase tripCreator
}

func New(usecase tripCreator) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req apiGen.TripData

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		domainerror.Write(w, r, http.StatusBadRequest,
			"invalid_request", "Invalid JSON body or unknown fields")
		return
	}

	if req.Price <= 0 || req.UserId == uuid.Nil || req.DriverId == uuid.Nil {
		domainerror.Write(w, r, http.StatusBadRequest,
			"invalid_request", "Invalid fields: price, user_id, driver_id are required")
		return
	}

	if req.StartPoint.Latitude < -90 || req.StartPoint.Latitude > 90 ||
		req.EndPoint.Latitude < -90 || req.EndPoint.Latitude > 90 ||
		req.StartPoint.Longitude < -180 || req.StartPoint.Longitude > 180 ||
		req.EndPoint.Longitude < -180 || req.EndPoint.Longitude > 180 {
		domainerror.Write(w, r, http.StatusBadRequest,
			"invalid_request", "Coordinates out of range")
		return
	}

	input := triphttp.ToCreateTripInput(req)

	trip, err := h.usecase.Execute(r.Context(), input)
	if err != nil {
		if errors.Is(err, domaintrip.ErrDriverBusy) {
			domainerror.Write(w, r, http.StatusConflict,
				"driver_busy", "Driver already has an active trip")
			return
		}

		slog.Error("failed to create trip", "error", err)
		domainerror.Write(w, r, http.StatusInternalServerError,
			"internal_error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(triphttp.ToAPITrip(trip))
}
