//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=get_test
package get

import (
	"context"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
