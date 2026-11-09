package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/http/trip/create"
	"github.com/marykorikova/Go/internal/model"
)

type fakeTripCreator struct {
	trip   model.Trip
	err    error
	called bool
}

func (f *fakeTripCreator) Execute(
	_ context.Context,
	_ model.CreateTripInput,
) (model.Trip, error) {
	f.called = true
	return f.trip, f.err
}

func TestCreateTrip(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	tripID := uuid.New()

	validBody := `{
		"user_id": "` + userID.String() + `",
		"driver_id": "` + driverID.String() + `",
		"start_point": {"latitude": 55.75, "longitude": 37.62},
		"end_point": {"latitude": 59.93, "longitude": 30.31},
		"price": 1450
	}`

	tests := []struct {
		name       string
		body       string
		err        error
		wantStatus int
		wantCode   string
		wantCall   bool
	}{
		{
			name:       "поездка создана",
			body:       validBody,
			wantStatus: http.StatusCreated,
			wantCall:   true,
		},
		{
			name:       "некорректный JSON",
			body:       "{invalid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "неверные обязательные поля",
			body:       `{"price": 0}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name: "координаты вне диапазона",
			body: `{
				"user_id": "` + userID.String() + `",
				"driver_id": "` + driverID.String() + `",
				"start_point": {"latitude": 100, "longitude": 37.62},
				"end_point": {"latitude": 59.93, "longitude": 30.31},
				"price": 1450
			}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "водитель занят",
			body:       validBody,
			err:        domaintrip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
			wantCall:   true,
		},
		{
			name:       "внутренняя ошибка",
			body:       validBody,
			err:        errors.New("database error"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
			wantCall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &fakeTripCreator{
				trip: model.Trip{
					ID:        tripID,
					UserID:    userID,
					DriverID:  driverID,
					Status:    domaintrip.StatusActive,
					StartedAt: time.Now(),
					Price:     1450,
				},
				err: tt.err,
			}

			handler := create.New(uc)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips",
				strings.NewReader(tt.body),
			)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if uc.called != tt.wantCall {
				t.Errorf("usecase called = %v, want %v", uc.called, tt.wantCall)
			}

			if tt.wantStatus == http.StatusCreated {
				if got := rec.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}

				wantLocation := "/api/v1/trips/" + tripID.String()
				if got := rec.Header().Get("Location"); got != wantLocation {
					t.Errorf("Location = %q, want %q", got, wantLocation)
				}

				var result apiGen.Trip
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if result.Id != tripID {
					t.Errorf("ID = %v, want %v", result.Id, tripID)
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
