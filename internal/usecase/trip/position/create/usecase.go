package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/telemetry"
)

type Usecase struct {
	metrics  *telemetry.PositionSavedMetrics
	repo     tripRepository
	txm      txManager
	canceler positionJobCanceler
}

func New(repo tripRepository, txm txManager, cancelers ...positionJobCanceler) *Usecase {
	u := &Usecase{
		repo: repo,
		txm:  txm,
	}

	if len(cancelers) > 0 {
		u.canceler = cancelers[0]
	}

	return u
}

func (u *Usecase) SetMetrics(metrics *telemetry.PositionSavedMetrics) {
	u.metrics = metrics
}

func (u *Usecase) Execute(
	ctx context.Context,
	tripID uuid.UUID,
	coordinates model.Coordinates,
	recordedAt time.Time,
) (model.TripPosition, error) {
	var result model.TripPosition

	err := u.txm.Do(ctx, func(txCtx context.Context) error {
		if err := u.repo.LockForPosition(txCtx, tripID); err != nil {
			return err
		}

		position, err := u.repo.SavePosition(
			txCtx,
			tripID,
			coordinates,
			recordedAt,
		)
		if err != nil {
			return err
		}

		result = position
		return nil
	})
	if err != nil {
		return model.TripPosition{}, err
	}

	if u.canceler != nil {
		u.canceler.Cancel(tripID)
	}

	if u.metrics != nil {
		u.metrics.RecordSaved(ctx)
	}

	return result, nil
}
