package finish

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
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
		slog.Error("failed to finish trip", "error", err)
		domainerror.WriteDomainError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(triphttp.ToAPITrip(trip))
}
