package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type tripRepository interface {
	LockForPosition(ctx context.Context, tripID uuid.UUID) error
	SavePosition(
		ctx context.Context,
		tripID uuid.UUID,
		coordinates model.Coordinates,
		recordedAt time.Time,
	) (model.TripPosition, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type positionJobCanceler interface {
	Cancel(tripID uuid.UUID)
}
