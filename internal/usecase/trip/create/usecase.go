package create

import (
	"context"
	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"go.opentelemetry.io/otel"
	"time"
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
	ctx, span := otel.Tracer("trip-service/usecase").Start(ctx, "trip.create")
	defer span.End()

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
