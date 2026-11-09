//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

package finish

import (
	"context"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, trip model.Trip) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
