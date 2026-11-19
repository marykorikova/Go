package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/usecase/trip/finish"
	"go.uber.org/mock/gomock"
)

func TestFinishTrip(t *testing.T) {
	dbErr := errors.New("database error")
	txErr := errors.New("transaction error")

	tests := []struct {
		name       string
		status     domaintrip.Status
		getErr     error
		finishErr  error
		txErr      error
		wantErr    error
		wantFinish bool
	}{
		{
			name:       "поездка успешно завершена",
			status:     domaintrip.StatusActive,
			wantFinish: true,
		},
		{
			name:    "поездка уже завершена",
			status:  domaintrip.StatusCompleted,
			wantErr: domaintrip.ErrTripCompleted,
		},
		{
			name:    "поездка не найдена",
			getErr:  domaintrip.ErrTripNotFound,
			wantErr: domaintrip.ErrTripNotFound,
		},
		{
			name:    "ошибка получения поездки",
			getErr:  dbErr,
			wantErr: dbErr,
		},
		{
			name:       "ошибка сохранения",
			status:     domaintrip.StatusActive,
			finishErr:  dbErr,
			wantErr:    dbErr,
			wantFinish: true,
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

			id := uuid.New()
			startedAt := time.Now().Add(-time.Hour)

			existing := model.Trip{
				ID:        id,
				Status:    tt.status,
				StartedAt: startedAt,
			}

			if tt.txErr == nil {
				repo.EXPECT().
					GetByID(gomock.Any(), id).
					Return(existing, tt.getErr).
					Times(1)
			}

			if tt.wantFinish {
				repo.EXPECT().
					Finish(gomock.Any(), id, gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						tripID uuid.UUID,
						trip model.Trip,
					) error {
						if tripID != id {
							t.Errorf("ID = %v, want %v", tripID, id)
						}
						if trip.Status != domaintrip.StatusCompleted {
							t.Error("статус должен быть completed")
						}
						if trip.FinishedAt == nil {
							t.Error("время завершения не установлено")
						} else if trip.FinishedAt.Before(startedAt) {
							t.Error("время завершения раньше начала")
						}
						return tt.finishErr
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

			uc := finish.New(repo, txm)
			result, err := uc.Execute(context.Background(), id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil {
				if result.ID != id {
					t.Errorf("ID = %v, want %v", result.ID, id)
				}
				if result.Status != domaintrip.StatusCompleted {
					t.Errorf("Status = %v, want completed", result.Status)
				}
				if result.FinishedAt == nil {
					t.Error("FinishedAt не установлен")
				}
			} else if result.ID != uuid.Nil {
				t.Error("при ошибке ожидается пустая поездка")
			}
		})
	}
}
