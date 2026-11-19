package list

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/model"
)

type fakePositionLister struct {
	positions []model.TripPosition
	err       error
}

func (f fakePositionLister) Execute(
	ctx context.Context,
	tripID uuid.UUID,
) ([]model.TripPosition, error) {
	return f.positions, f.err
}

func TestHandler_OK(t *testing.T) {
	tripID := uuid.New()
	recordedAt := time.Date(2026, 8, 24, 12, 0, 5, 0, time.UTC)

	fake := fakePositionLister{
		positions: []model.TripPosition{
			{
				ID:          42,
				TripID:      tripID,
				Coordinates: model.Coordinates{Latitude: 59.935, Longitude: 30.335},
				RecordedAt:  recordedAt,
			},
		},
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String()+"/positions",
		nil,
	)
	recorder := httptest.NewRecorder()

	New(fake).Handle(recorder, request, tripID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s",
			recorder.Code, recorder.Body.String())
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var response apiGen.TripPositionsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Positions) != 1 {
		t.Fatalf("positions count = %d, want 1", len(response.Positions))
	}

	position := response.Positions[0]

	if position.Id != 42 {
		t.Errorf("position ID = %d, want 42", position.Id)
	}
	if position.TripId != tripID {
		t.Errorf("trip ID = %s, want %s", position.TripId, tripID)
	}
	if position.Coordinates.Latitude != 59.935 ||
		position.Coordinates.Longitude != 30.335 {
		t.Errorf("unexpected coordinates: %+v", position.Coordinates)
	}
	if !position.RecordedAt.Equal(recordedAt) {
		t.Errorf("recorded_at = %s, want %s", position.RecordedAt, recordedAt)
	}
}

func TestHandler_EmptyPositions(t *testing.T) {
	tripID := uuid.New()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String()+"/positions",
		nil,
	)
	recorder := httptest.NewRecorder()

	New(fakePositionLister{
		positions: []model.TripPosition{},
	}).Handle(recorder, request, tripID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s",
			recorder.Code, recorder.Body.String())
	}

	var response struct {
		Positions []json.RawMessage `json:"positions"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Positions == nil {
		t.Fatal("positions is null, want empty array []")
	}

	if len(response.Positions) != 0 {
		t.Errorf("positions count = %d, want 0", len(response.Positions))
	}
}

func TestHandler_TripNotFound(t *testing.T) {
	tripID := uuid.New()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String()+"/positions",
		nil,
	)
	recorder := httptest.NewRecorder()

	New(fakePositionLister{
		err: domaintrip.ErrTripNotFound,
	}).Handle(recorder, request, tripID)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s",
			recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code string `json:"code"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Code != "trip_not_found" {
		t.Errorf("code = %q, want trip_not_found", response.Code)
	}
}
