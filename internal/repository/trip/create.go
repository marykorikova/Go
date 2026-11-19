package trip

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
)

func (r *Repository) Create(ctx context.Context, trip model.Trip) error {
	exec := r.getExecutor(ctx)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
		)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "trips_driver_active_idx" {
			return domaintrip.ErrDriverBusy
		}

		return err
	}

	historyQuery := psql.Insert("trip_status_history").
		Columns("trip_id", "to_status", "reason").
		Values(trip.ID, trip.Status, "created")

	sqlHist, argsHist, err := historyQuery.ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, sqlHist, argsHist...)
	return err
}
