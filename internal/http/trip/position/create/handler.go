package create

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/domainerror"
	"github.com/marykorikova/Go/internal/model"
)

type positionCreator interface {
	Execute(
		ctx context.Context,
		tripID uuid.UUID,
		coordinates model.Coordinates,
		recordedAt time.Time,
	) (model.TripPosition, error)
}

type Handler struct {
	usecase positionCreator
}

func New(usecase positionCreator) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID) {
	var req struct {
		Coordinates *apiGen.Coordinates `json:"coordinates"`
		RecordedAt  *time.Time          `json:"recorded_at"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		domainerror.Write(w, r, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		domainerror.Write(w, r, http.StatusBadRequest, "invalid_request", "Expected one JSON object")
		return
	}

	if req.Coordinates == nil || req.RecordedAt == nil || req.RecordedAt.IsZero() {
		domainerror.Write(w, r, http.StatusBadRequest, "invalid_request", "Missing required fields")
		return
	}

	lat := req.Coordinates.Latitude
	lon := req.Coordinates.Longitude

	if math.IsNaN(lat) || math.IsInf(lat, 0) ||
		math.IsNaN(lon) || math.IsInf(lon, 0) ||
		lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		domainerror.Write(w, r, http.StatusBadRequest, "invalid_request", "Coordinates out of range")
		return
	}

	if req.RecordedAt.After(time.Now().Add(time.Minute)) {
		domainerror.Write(w, r, http.StatusBadRequest, "invalid_request", "Recorded time is too far in the future")
		return
	}

	position, err := h.usecase.Execute(
		r.Context(),
		tripID,
		model.Coordinates{Latitude: lat, Longitude: lon},
		*req.RecordedAt,
	)
	if err != nil {
		slog.Error("failed to save trip position", "error", err)
		domainerror.WriteDomainError(w, r, err)
		return
	}

	response := apiGen.TripPosition{
		Id:     position.ID,
		TripId: position.TripID,
		Coordinates: apiGen.Coordinates{
			Latitude:  position.Coordinates.Latitude,
			Longitude: position.Coordinates.Longitude,
		},
		RecordedAt: position.RecordedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}
