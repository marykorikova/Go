package get

import (
	"context"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type Usecase struct {
	repo tripRepository
}

func New(repo tripRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(
	ctx context.Context,
	id uuid.UUID,
) (model.Trip, error) {
	return u.repo.GetByID(ctx, id)
}
