package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/marykorikova/Go/internal/generated"
	"github.com/marykorikova/Go/internal/txmanager"
)

var (
	ErrNotFound     = errors.New("trip not found")
	ErrDriverBusy   = errors.New("driver busy")
	ErrTripFinished = errors.New("trip already finished")
)

type TripRepository struct {
	pool txmanager.Executor
}

func NewTripRepository(pool txmanager.Executor) *TripRepository {
	return &TripRepository{pool: pool}
}


func (r *TripRepository) getExecutor(ctx context.Context) txmanager.Executor {
	if tx := txmanager.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *TripRepository) Create(ctx context.Context, trip api.Trip) error {
	exec := r.getExecutor(ctx)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(trip.Id, trip.UserId, trip.DriverId, trip.StartPoint.Latitude, trip.StartPoint.Longitude, trip.EndPoint.Latitude, trip.EndPoint.Longitude, trip.Price, trip.Status, trip.StartedAt)

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Нарушение уникальности
			return ErrDriverBusy
		}
		return err
	}

	historyQuery := psql.Insert("trip_status_history").
		Columns("trip_id", "to_status", "reason").
		Values(trip.Id, trip.Status, "created")

	sqlHist, argsHist, _ := historyQuery.ToSql()
	_, err = exec.Exec(ctx, sqlHist, argsHist...)
	return err
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	exec := r.getExecutor(ctx)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Select("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(squirrel.Eq{"id": id})

	sql, args, _ := query.ToSql()

	var trip api.Trip
	err := exec.QueryRow(ctx, sql, args...).Scan(
		&trip.Id, &trip.UserId, &trip.DriverId,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude, &trip.EndPoint.Longitude,
		&trip.Price, &trip.Status, &trip.StartedAt, &trip.FinishedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return api.Trip{}, ErrNotFound
	}
	return trip, err
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	exec := r.getExecutor(ctx)
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query := psql.Update("trips").
		Set("status", "completed").
		Set("finished_at", time.Now()).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": id, "status": "active"}).
		Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at")

	sql, args, _ := query.ToSql()

	var trip api.Trip
	err := exec.QueryRow(ctx, sql, args...).Scan(
		&trip.Id, &trip.UserId, &trip.DriverId,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude, &trip.EndPoint.Longitude,
		&trip.Price, &trip.Status, &trip.StartedAt, &trip.FinishedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		_, checkErr := r.GetByID(ctx, id)
		if errors.Is(checkErr, ErrNotFound) {
			return api.Trip{}, ErrNotFound
		}
		return api.Trip{}, ErrTripFinished
	}

	if err == nil {
		historyQuery := psql.Insert("trip_status_history").
			Columns("trip_id", "from_status", "to_status", "reason").
			Values(id, "active", "completed", "finished by user")
		sqlHist, argsHist, _ := historyQuery.ToSql()
		_, _ = exec.Exec(ctx, sqlHist, argsHist...)
	}

	return trip, err
}
