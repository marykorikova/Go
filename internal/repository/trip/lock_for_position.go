package trip

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
)

func (r *Repository) LockForPosition(ctx context.Context, tripID uuid.UUID) error {
	var status domaintrip.Status

	err := r.getExecutor(ctx).QueryRow(
		ctx,
		"SELECT status FROM trips WHERE id = $1 FOR UPDATE",
		tripID,
	).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return domaintrip.ErrTripNotFound
	}
	if err != nil {
		return err
	}
	if status != domaintrip.StatusActive {
		return domaintrip.ErrTripCompleted
	}

	return nil
}
