package trip

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
)

func (r *Repository) Finish(
	ctx context.Context,
	id uuid.UUID,
	trip model.Trip,
) error {
	exec := r.getExecutor(ctx)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Update("trips").
		Set("status", trip.Status).
		Set("finished_at", trip.FinishedAt).
		Set("updated_at", trip.FinishedAt).
		Where(squirrel.Eq{
			"id":     id,
			"status": domaintrip.StatusActive,
		}).
		Suffix("RETURNING id")

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	var updatedID uuid.UUID
	err = exec.QueryRow(ctx, sql, args...).Scan(&updatedID)

	if errors.Is(err, pgx.ErrNoRows) {
		_, checkErr := r.GetByID(ctx, id)

		if errors.Is(checkErr, domaintrip.ErrTripNotFound) {
			return domaintrip.ErrTripNotFound
		}
		if checkErr != nil {
			return checkErr
		}

		return domaintrip.ErrTripCompleted
	}

	if err != nil {
		return err
	}

	historyQuery := psql.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(
			id,
			domaintrip.StatusActive,
			domaintrip.StatusCompleted,
			"finished by user",
		)

	historySQL, historyArgs, err := historyQuery.ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, historySQL, historyArgs...)
	return err
}
