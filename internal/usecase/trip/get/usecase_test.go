package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/usecase/trip/get"
	"go.uber.org/mock/gomock"
)

func TestGetTrip(t *testing.T) {
	tripID := uuid.New()
	dbErr := errors.New("database error")

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "поездка найдена",
		},
		{
			name:    "поездка не найдена",
			repoErr: domaintrip.ErrTripNotFound,
			wantErr: domaintrip.ErrTripNotFound,
		},
		{
			name:    "ошибка базы данных",
			repoErr: dbErr,
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)

			expectedTrip := model.Trip{
				ID:     tripID,
				Status: domaintrip.StatusActive,
			}

			repo.EXPECT().
				GetByID(gomock.Any(), tripID).
				Return(expectedTrip, tt.repoErr).
				Times(1)

			usecase := get.New(repo)

			result, err := usecase.Execute(
				context.Background(),
				tripID,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Execute() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil && result != expectedTrip {
				t.Errorf(
					"Execute() result = %+v, want %+v",
					result,
					expectedTrip,
				)
			}
		})
	}
}
