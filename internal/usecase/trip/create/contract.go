//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=create_test

package create

import (
	"context"

	"github.com/marykorikova/Go/internal/model"
)

type tripRepository interface {
	Create(ctx context.Context, trip model.Trip) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
