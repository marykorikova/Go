package finish

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/http/domainerror"
	triphttp "github.com/marykorikova/Go/internal/http/trip"
	"github.com/marykorikova/Go/internal/model"
)

type tripFinisher interface {
	Execute(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

type Handler struct {
	usecase tripFinisher
}

func New(usecase tripFinisher) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	trip, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, domaintrip.ErrTripNotFound) {
			domainerror.Write(w, r, http.StatusNotFound,
				"trip_not_found", "Trip not found")
			return
		}

		if errors.Is(err, domaintrip.ErrTripCompleted) {
			domainerror.Write(w, r, http.StatusConflict,
				"trip_completed", "Trip is already completed")
			return
		}

		slog.Error("failed to finish trip", "error", err)
		domainerror.Write(w, r, http.StatusInternalServerError,
			"internal_error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(triphttp.ToAPITrip(trip))
}
