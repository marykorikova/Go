package list

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/domainerror"
	"github.com/marykorikova/Go/internal/model"
)

type positionLister interface {
	Execute(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error)
}

type Handler struct {
	usecase positionLister
}

func New(usecase positionLister) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID) {
	positions, err := h.usecase.Execute(r.Context(), tripID)
	if err != nil {
		slog.Error("failed to list trip positions", "error", err)
		domainerror.WriteDomainError(w, r, err)
		return
	}

	response := apiGen.TripPositionsResponse{
		Positions: make([]apiGen.TripPosition, 0, len(positions)),
	}

	for _, position := range positions {
		response.Positions = append(response.Positions, apiGen.TripPosition{
			Id:     position.ID,
			TripId: position.TripID,
			Coordinates: apiGen.Coordinates{
				Latitude:  position.Coordinates.Latitude,
				Longitude: position.Coordinates.Longitude,
			},
			RecordedAt: position.RecordedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
