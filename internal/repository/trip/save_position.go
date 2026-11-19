package trip

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

func (r *Repository) SavePosition(
	ctx context.Context,
	tripID uuid.UUID,
	coordinates model.Coordinates,
	recordedAt time.Time,
) (model.TripPosition, error) {
	exec := r.getExecutor(ctx)

	position := model.TripPosition{
		TripID:      tripID,
		Coordinates: coordinates,
		RecordedAt:  recordedAt,
	}

	err := exec.QueryRow(ctx, `
		INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, tripID, coordinates.Latitude, coordinates.Longitude, recordedAt).Scan(&position.ID)
	if err != nil {
		return model.TripPosition{}, err
	}

	_, err = exec.Exec(ctx, `
		UPDATE trips
		SET last_position_at = GREATEST(
			COALESCE(last_position_at, $1),
			$1
		)
		WHERE id = $2
	`, recordedAt, tripID)
	if err != nil {
		return model.TripPosition{}, err
	}

	return position, nil
}
