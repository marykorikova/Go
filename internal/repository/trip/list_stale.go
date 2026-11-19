package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type StaleTrip struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
}

func (r *Repository) ListStale(ctx context.Context, staleBefore time.Time) ([]StaleTrip, error) {
	rows, err := r.getExecutor(ctx).Query(ctx, `
		SELECT id, driver_id
		FROM trips
		WHERE status = 'active'
		  AND COALESCE(last_position_at, started_at) < $1
		ORDER BY COALESCE(last_position_at, started_at) ASC, id ASC
	`, staleBefore)
	if err != nil {
		return nil, fmt.Errorf("query stale trips: %w", err)
	}
	defer rows.Close()

	trips := make([]StaleTrip, 0)
	for rows.Next() {
		var trip StaleTrip
		if err := rows.Scan(&trip.TripID, &trip.DriverID); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale trips: %w", err)
	}
	return trips, nil
}
