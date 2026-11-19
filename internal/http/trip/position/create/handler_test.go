package create

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
)

type fakePositionCreator struct {
	position model.TripPosition
	err      error
}

func (f fakePositionCreator) Execute(
	ctx context.Context,
	tripID uuid.UUID,
	coordinates model.Coordinates,
	recordedAt time.Time,
) (model.TripPosition, error) {
	return f.position, f.err
}

func TestHandler_Created(t *testing.T) {
	tripID := uuid.New()
	recordedAt := time.Date(2026, 8, 24, 12, 0, 5, 0, time.UTC)

	usecase := fakePositionCreator{
		position: model.TripPosition{
			ID:          42,
			TripID:      tripID,
			Coordinates: model.Coordinates{Latitude: 59.935, Longitude: 30.335},
			RecordedAt:  recordedAt,
		},
	}

	body := `{"coordinates":{"latitude":59.935,"longitude":30.335},"recorded_at":"2026-08-24T12:00:05Z"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/positions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	New(usecase).Handle(recorder, request, tripID)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	if !strings.Contains(recorder.Body.String(), `"id":42`) {
		t.Errorf("response does not contain position ID: %s", recorder.Body.String())
	}
}

func TestHandler_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "latitude out of range",
			body: `{"coordinates":{"latitude":91,"longitude":30},"recorded_at":"2026-08-24T12:00:05Z"}`,
		},
		{
			name: "longitude out of range",
			body: `{"coordinates":{"latitude":59,"longitude":181},"recorded_at":"2026-08-24T12:00:05Z"}`,
		},
		{
			name: "invalid timestamp",
			body: `{"coordinates":{"latitude":59,"longitude":30},"recorded_at":"not-a-date"}`,
		},
		{
			name: "missing coordinates",
			body: `{"recorded_at":"2026-08-24T12:00:05Z"}`,
		},
		{
			name: "missing timestamp",
			body: `{"coordinates":{"latitude":59,"longitude":30}}`,
		},
		{
			name: "invalid JSON",
			body: `{"coordinates":`,
		},
		{
			name: "unknown field",
			body: `{"coordinates":{"latitude":59,"longitude":30},"recorded_at":"2026-08-24T12:00:05Z","unexpected":true}`,
		},
		{
			name: "timestamp too far in future",
			body: `{"coordinates":{"latitude":59,"longitude":30},"recorded_at":"2999-01-01T00:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tripID := uuid.New()
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips/"+tripID.String()+"/positions",
				strings.NewReader(tt.body),
			)
			recorder := httptest.NewRecorder()

			New(fakePositionCreator{}).Handle(recorder, request, tripID)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s",
					recorder.Code, recorder.Body.String())
			}

			if !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
				t.Errorf("expected invalid_request; body = %s",
					recorder.Body.String())
			}
		})
	}
}

func TestHandler_DomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "trip not found",
			err:        domaintrip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "trip completed",
			err:        domaintrip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tripID := uuid.New()
			body := `{"coordinates":{"latitude":59.935,"longitude":30.335},"recorded_at":"2026-08-24T12:00:05Z"}`

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips/"+tripID.String()+"/positions",
				strings.NewReader(body),
			)
			recorder := httptest.NewRecorder()

			New(fakePositionCreator{err: tt.err}).Handle(recorder, request, tripID)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s",
					recorder.Code, tt.wantStatus, recorder.Body.String())
			}

			if !strings.Contains(recorder.Body.String(), `"code":"`+tt.wantCode+`"`) {
				t.Errorf("expected code %q; body = %s",
					tt.wantCode, recorder.Body.String())
			}
		})
	}
}
