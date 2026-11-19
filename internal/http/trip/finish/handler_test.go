package finish_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/trip/finish"
	"github.com/marykorikova/Go/internal/model"
)

type fakeTripFinisher struct {
	trip   model.Trip
	err    error
	called bool
	gotID  uuid.UUID
}

func (f *fakeTripFinisher) Execute(
	_ context.Context,
	id uuid.UUID,
) (model.Trip, error) {
	f.called = true
	f.gotID = id
	return f.trip, f.err
}

func TestFinishTrip(t *testing.T) {
	id := uuid.New()
	finishedAt := time.Now()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "поездка завершена",
			wantStatus: http.StatusOK,
		},
		{
			name:       "поездка не найдена",
			err:        domaintrip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "поездка уже завершена",
			err:        domaintrip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "внутренняя ошибка",
			err:        errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeTripFinisher{
				trip: model.Trip{
					ID:         id,
					Status:     domaintrip.StatusCompleted,
					FinishedAt: &finishedAt,
				},
				err: tt.err,
			}

			handler := finish.New(uc)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips/"+id.String()+"/finish",
				nil,
			)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req, id)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if !uc.called {
				t.Fatal("usecase was not called")
			}

			if uc.gotID != id {
				t.Errorf("usecase ID = %v, want %v", uc.gotID, id)
			}

			if tt.wantStatus == http.StatusOK {
				if got := rec.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}

				var result apiGen.Trip
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if result.Id != id {
					t.Errorf("ID = %v, want %v", result.Id, id)
				}
				if result.Status != apiGen.Completed {
					t.Errorf("Status = %v, want completed", result.Status)
				}
				if result.FinishedAt == nil {
					t.Error("FinishedAt is nil")
				}
				return
			}

			if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Errorf("Content-Type = %q, want application/problem+json", got)
			}

			var problem struct {
				Status int    `json:"status"`
				Code   string `json:"code"`
			}

			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
				t.Fatalf("invalid error JSON: %v", err)
			}

			if problem.Status != tt.wantStatus {
				t.Errorf(
					"problem status = %d, want %d",
					problem.Status,
					tt.wantStatus,
				)
			}

			if problem.Code != tt.wantCode {
				t.Errorf(
					"problem code = %q, want %q",
					problem.Code,
					tt.wantCode,
				)
			}
		})
	}
}
