package trip

import (
	"testing"
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/model"
)

func TestToCreateTripInput(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()

	request := apiGen.TripData{
		UserId:   userID,
		DriverId: driverID,
		StartPoint: apiGen.Coordinates{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: apiGen.Coordinates{
			Latitude:  59.93,
			Longitude: 30.31,
		},
		Price: 1500,
	}

	result := ToCreateTripInput(request)

	if result.UserID != userID {
		t.Errorf("UserID = %v, want %v", result.UserID, userID)
	}

	if result.DriverID != driverID {
		t.Errorf("DriverID = %v, want %v", result.DriverID, driverID)
	}

	if result.StartPoint.Latitude != 55.75 ||
		result.StartPoint.Longitude != 37.61 {
		t.Errorf("unexpected StartPoint: %+v", result.StartPoint)
	}

	if result.EndPoint.Latitude != 59.93 ||
		result.EndPoint.Longitude != 30.31 {
		t.Errorf("unexpected EndPoint: %+v", result.EndPoint)
	}

	if result.Price != 1500 {
		t.Errorf("Price = %d, want 1500", result.Price)
	}
}

func TestToAPITrip(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()

	startedAt := time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(30 * time.Minute)

	trip := model.Trip{
		ID:       id,
		UserID:   userID,
		DriverID: driverID,
		StartPoint: model.Coordinates{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: model.Coordinates{
			Latitude:  59.93,
			Longitude: 30.31,
		},
		Price:      1500,
		Status:     domaintrip.StatusCompleted,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
	}

	result := ToAPITrip(trip)

	if result.Id != id {
		t.Errorf("Id = %v, want %v", result.Id, id)
	}

	if result.UserId != userID {
		t.Errorf("UserId = %v, want %v", result.UserId, userID)
	}

	if result.DriverId != driverID {
		t.Errorf("DriverId = %v, want %v", result.DriverId, driverID)
	}

	if result.StartPoint.Latitude != 55.75 ||
		result.StartPoint.Longitude != 37.61 {
		t.Errorf("unexpected StartPoint: %+v", result.StartPoint)
	}

	if result.EndPoint.Latitude != 59.93 ||
		result.EndPoint.Longitude != 30.31 {
		t.Errorf("unexpected EndPoint: %+v", result.EndPoint)
	}

	if result.Price != 1500 {
		t.Errorf("Price = %d, want 1500", result.Price)
	}

	if result.Status != apiGen.TripStatus(domaintrip.StatusCompleted) {
		t.Errorf("Status = %v, want completed", result.Status)
	}

	if !result.StartedAt.Equal(startedAt) {
		t.Errorf("StartedAt = %v, want %v", result.StartedAt, startedAt)
	}

	if result.FinishedAt == nil || !result.FinishedAt.Equal(finishedAt) {
		t.Errorf("FinishedAt = %v, want %v", result.FinishedAt, finishedAt)
	}

	if result.LastPositionAt != nil {
		t.Errorf("LastPositionAt = %v, want nil", result.LastPositionAt)
	}
}
