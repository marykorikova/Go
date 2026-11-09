package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
)

type Usecase struct {
	repo tripRepository
	txm  txManager
}

func New(repo tripRepository, txm txManager) *Usecase {
	return &Usecase{
		repo: repo,
		txm:  txm,
	}
}

func (u *Usecase) Execute(
	ctx context.Context,
	input model.CreateTripInput,
) (model.Trip, error) {
	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now(),
	}

	err := u.txm.Do(ctx, func(txCtx context.Context) error {
		return u.repo.Create(txCtx, trip)
	})
	if err != nil {
		return model.Trip{}, err
	}

	return trip, nil
}
