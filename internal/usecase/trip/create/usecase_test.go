package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/usecase/trip/create"
	"go.uber.org/mock/gomock"
)

func TestCreateTrip(t *testing.T) {
	dbErr := errors.New("database error")
	txErr := errors.New("transaction error")

	tests := []struct {
		name     string
		repoErr  error
		txErr    error
		wantErr  error
		callRepo bool
	}{
		{
			name:     "поездка успешно создана",
			callRepo: true,
		},
		{
			name:     "водитель занят",
			repoErr:  domaintrip.ErrDriverBusy,
			wantErr:  domaintrip.ErrDriverBusy,
			callRepo: true,
		},
		{
			name:     "ошибка базы данных",
			repoErr:  dbErr,
			wantErr:  dbErr,
			callRepo: true,
		},
		{
			name:    "ошибка транзакции",
			txErr:   txErr,
			wantErr: txErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := NewMocktripRepository(ctrl)
			txm := NewMocktxManager(ctrl)

			input := model.CreateTripInput{
				UserID:   uuid.New(),
				DriverID: uuid.New(),
				StartPoint: model.Coordinates{
					Latitude:  55.75,
					Longitude: 37.61,
				},
				EndPoint: model.Coordinates{
					Latitude:  59.93,
					Longitude: 30.31,
				},
				Price: 1000,
			}

			before := time.Now()

			if tt.callRepo {
				repo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						trip model.Trip,
					) error {
						if trip.ID == uuid.Nil {
							t.Error("ID поездки не задан")
						}
						if trip.UserID != input.UserID {
							t.Error("неверный UserID")
						}
						if trip.DriverID != input.DriverID {
							t.Error("неверный DriverID")
						}
						if trip.StartPoint != input.StartPoint {
							t.Error("неверная начальная точка")
						}
						if trip.EndPoint != input.EndPoint {
							t.Error("неверная конечная точка")
						}
						if trip.Price != input.Price {
							t.Error("неверная стоимость")
						}
						if trip.Status != domaintrip.StatusActive {
							t.Error("неверный статус")
						}
						if trip.StartedAt.Before(before) ||
							trip.StartedAt.After(time.Now()) {
							t.Error("неверное время начала")
						}

						return tt.repoErr
					}).
					Times(1)
			}

			txm.EXPECT().
				Do(gomock.Any(), gomock.Any()).
				DoAndReturn(func(
					ctx context.Context,
					fn func(context.Context) error,
				) error {
					if tt.txErr != nil {
						return tt.txErr
					}
					return fn(ctx)
				}).
				Times(1)

			uc := create.New(repo, txm)

			result, err := uc.Execute(
				context.Background(),
				input,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Execute() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil {
				if result.ID == uuid.Nil {
					t.Error("результат не содержит ID")
				}
				if result.UserID != input.UserID {
					t.Error("неверный UserID результата")
				}
				if result.Status != domaintrip.StatusActive {
					t.Error("неверный статус результата")
				}
			} else if result.ID != uuid.Nil {
				t.Error("при ошибке ожидается пустая поездка")
			}
		})
	}
}
