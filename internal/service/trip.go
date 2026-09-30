package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/repository"
	"github.com/marykorikova/Go/internal/txmanager"
)

type TripService struct {
	repo *repository.TripRepository
	txm  txmanager.TxManager
}

func NewTripService(repo *repository.TripRepository, txm txmanager.TxManager) *TripService {
	return &TripService{repo: repo, txm: txm}
}

func (s *TripService) CreateTrip(ctx context.Context, req api.Trip) (api.Trip, error) {
	req.Id = uuid.New()
	req.Status = "active"
	req.StartedAt = time.Now()

	err := s.txm.Do(ctx, func(ctx context.Context) error {
		return s.repo.Create(ctx, req)
	})
	if err != nil {
		return api.Trip{}, err
	}
	return req, nil
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	var trip api.Trip
	err := s.txm.Do(ctx, func(ctx context.Context) error {
		var err error
		trip, err = s.repo.Finish(ctx, id)
		return err
	})
	return trip, err
}
