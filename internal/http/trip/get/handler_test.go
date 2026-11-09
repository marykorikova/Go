package get_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/trip/get"
	"github.com/marykorikova/Go/internal/model"
)

type fakeTripGetter struct {
	trip model.Trip
	err  error
}

func (f *fakeTripGetter) Execute(
	_ context.Context,
	_ uuid.UUID,
) (model.Trip, error) {
	return f.trip, f.err
}

func TestGetTrip(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name       string
		trip       model.Trip
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name: "поездка найдена",
			trip: model.Trip{
				ID:     id,
				Status: domaintrip.StatusActive,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "поездка не найдена",
			err:        domaintrip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
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
			uc := &fakeTripGetter{
				trip: tt.trip,
				err:  tt.err,
			}
			handler := get.New(uc)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/trips/"+id.String(),
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
				if result.Status != apiGen.Active {
					t.Errorf("Status = %v, want active", result.Status)
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
				t.Errorf("problem status = %d, want %d", problem.Status, tt.wantStatus)
			}
			if problem.Code != tt.wantCode {
				t.Errorf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}
		})
	}
}
