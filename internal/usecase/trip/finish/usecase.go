package finish

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
	id uuid.UUID,
) (model.Trip, error) {
	var result model.Trip

	err := u.txm.Do(ctx, func(txCtx context.Context) error {
		current, err := u.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		domain := domaintrip.Trip{
			Status:     current.Status,
			FinishedAt: current.FinishedAt,
		}

		if err := domain.Finish(time.Now()); err != nil {
			return err
		}

		current.Status = domain.Status
		current.FinishedAt = domain.FinishedAt

		if err := u.repo.Finish(txCtx, id, current); err != nil {
			return err
		}

		result = current
		return nil
	})
	if err != nil {
		return model.Trip{}, err
	}

	return result, nil
}
