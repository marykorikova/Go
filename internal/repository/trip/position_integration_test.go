//go:build integration

package trip

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/txmanager"
)

func TestSavePosition_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      500,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().Add(-time.Minute),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM trips WHERE id = $1", trip.ID)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	if err := txm.Do(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, trip)
	}); err != nil {
		t.Fatal(err)
	}

	recordedAt := time.Now().UTC().Truncate(time.Microsecond)
	coordinates := model.Coordinates{Latitude: 59.935, Longitude: 30.335}

	err = txm.Do(ctx, func(txCtx context.Context) error {
		if err := repo.LockForPosition(txCtx, trip.ID); err != nil {
			return err
		}

		_, err := repo.SavePosition(txCtx, trip.ID, coordinates, recordedAt)
		return err
	})
	if err != nil {
		t.Fatalf("save position: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM trip_positions WHERE trip_id = $1",
		trip.ID,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("positions count = %d, want 1", count)
	}

	var lastPositionAt time.Time
	err = pool.QueryRow(ctx,
		"SELECT last_position_at FROM trips WHERE id = $1",
		trip.ID,
	).Scan(&lastPositionAt)
	if err != nil {
		t.Fatal(err)
	}
	if !lastPositionAt.Equal(recordedAt) {
		t.Errorf("last_position_at = %v, want %v", lastPositionAt, recordedAt)
	}
}

func TestSavePosition_Rollback_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      500,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().Add(-time.Minute),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM trips WHERE id = $1", trip.ID)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	if err := txm.Do(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, trip)
	}); err != nil {
		t.Fatal(err)
	}

	rollbackErr := errors.New("forced rollback")

	err = txm.Do(ctx, func(txCtx context.Context) error {
		if err := repo.LockForPosition(txCtx, trip.ID); err != nil {
			return err
		}

		_, err := repo.SavePosition(
			txCtx,
			trip.ID,
			model.Coordinates{Latitude: 59.935, Longitude: 30.335},
			time.Now().UTC(),
		)
		if err != nil {
			return err
		}

		return rollbackErr
	})

	if !errors.Is(err, rollbackErr) {
		t.Fatalf("got error %v, want forced rollback", err)
	}

	var count int
	var lastPositionAt *time.Time

	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM trip_positions WHERE trip_id = $1),
			last_position_at
		FROM trips
		WHERE id = $1
	`, trip.ID).Scan(&count, &lastPositionAt)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Errorf("positions count = %d, want 0", count)
	}
	if lastPositionAt != nil {
		t.Errorf("last_position_at = %v, want NULL", lastPositionAt)
	}
}

func TestSavePosition_OlderPosition_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      500,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().Add(-time.Hour),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM trips WHERE id = $1", trip.ID)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	if err := txm.Do(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, trip)
	}); err != nil {
		t.Fatal(err)
	}

	newerTime := time.Now().UTC().Truncate(time.Microsecond)
	olderTime := newerTime.Add(-5 * time.Minute)

	for _, recordedAt := range []time.Time{newerTime, olderTime} {
		err := txm.Do(ctx, func(txCtx context.Context) error {
			if err := repo.LockForPosition(txCtx, trip.ID); err != nil {
				return err
			}

			_, err := repo.SavePosition(
				txCtx,
				trip.ID,
				model.Coordinates{Latitude: 59.935, Longitude: 30.335},
				recordedAt,
			)
			return err
		})
		if err != nil {
			t.Fatalf("save position at %v: %v", recordedAt, err)
		}
	}

	var count int
	var lastPositionAt time.Time

	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM trip_positions WHERE trip_id = $1),
			last_position_at
		FROM trips
		WHERE id = $1
	`, trip.ID).Scan(&count, &lastPositionAt)
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Errorf("positions count = %d, want 2", count)
	}
	if !lastPositionAt.Equal(newerTime) {
		t.Errorf("last_position_at = %v, want %v", lastPositionAt, newerTime)
	}
}

func TestSavePosition_CompletedTrip_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      500,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().Add(-time.Hour),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM trips WHERE id = $1", trip.ID)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	if err := txm.Do(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, trip)
	}); err != nil {
		t.Fatal(err)
	}

	trip.Status = domaintrip.StatusCompleted
	finishedAt := time.Now().UTC()
	trip.FinishedAt = &finishedAt

	if err := txm.Do(ctx, func(txCtx context.Context) error {
		return repo.Finish(txCtx, trip.ID, trip)
	}); err != nil {
		t.Fatalf("finish trip: %v", err)
	}

	err = txm.Do(ctx, func(txCtx context.Context) error {
		if err := repo.LockForPosition(txCtx, trip.ID); err != nil {
			return err
		}

		_, err := repo.SavePosition(
			txCtx,
			trip.ID,
			model.Coordinates{Latitude: 59.935, Longitude: 30.335},
			time.Now().UTC(),
		)
		return err
	})

	if !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Fatalf("got error %v, want ErrTripCompleted", err)
	}

	var count int
	err = pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM trip_positions WHERE trip_id = $1",
		trip.ID,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Errorf("positions count = %d, want 0", count)
	}
}
