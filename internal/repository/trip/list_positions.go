package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
)

func (r *Repository) ListPositions(
	ctx context.Context,
	tripID uuid.UUID,
) ([]model.TripPosition, error) {
	exec := r.getExecutor(ctx)

	var exists bool
	err := exec.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM trips WHERE id = $1)",
		tripID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check trip existence: %w", err)
	}
	if !exists {
		return nil, domaintrip.ErrTripNotFound
	}

	rows, err := exec.Query(ctx, `
		SELECT id, latitude, longitude, recorded_at
		FROM trip_positions
		WHERE trip_id = $1
		ORDER BY recorded_at ASC, id ASC
	`, tripID)
	if err != nil {
		return nil, fmt.Errorf("query trip positions: %w", err)
	}
	defer rows.Close()

	positions := make([]model.TripPosition, 0)

	for rows.Next() {
		var position model.TripPosition
		position.TripID = tripID

		err := rows.Scan(
			&position.ID,
			&position.Coordinates.Latitude,
			&position.Coordinates.Longitude,
			&position.RecordedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trip position: %w", err)
		}

		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domaintrip.ErrTripNotFound
		}
		return nil, fmt.Errorf("iterate trip positions: %w", err)
	}

	return positions, nil
}
