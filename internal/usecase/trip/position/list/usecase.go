package list

import (
	"context"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type positionRepository interface {
	ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error)
}

type Usecase struct {
	repo positionRepository
}

func New(repo positionRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(
	ctx context.Context,
	tripID uuid.UUID,
) ([]model.TripPosition, error) {
	return u.repo.ListPositions(ctx, tripID)
}
