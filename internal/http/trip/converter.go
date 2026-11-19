package trip

import (
	apiGen "github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/model"
)

func ToCreateTripInput(req apiGen.TripData) model.CreateTripInput {
	return model.CreateTripInput{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		StartPoint: model.Coordinates{
			Latitude:  req.StartPoint.Latitude,
			Longitude: req.StartPoint.Longitude,
		},
		EndPoint: model.Coordinates{
			Latitude:  req.EndPoint.Latitude,
			Longitude: req.EndPoint.Longitude,
		},
		Price: req.Price,
	}
}

func ToAPITrip(trip model.Trip) apiGen.Trip {
	return apiGen.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: apiGen.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: apiGen.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:          trip.Price,
		Status:         apiGen.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: trip.LastPositionAt,
	}
}
