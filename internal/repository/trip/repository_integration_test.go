//go:build integration

package trip

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/txmanager"
	"os"
	"testing"
	"time"
)

func TestRepository_Create_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		StartPoint: model.Coordinates{
			Latitude:  55.75,
			Longitude: 37.62,
		},
		EndPoint: model.Coordinates{
			Latitude:  55.76,
			Longitude: 37.63,
		},
		Price:     500,
		Status:    domaintrip.StatusActive,
		StartedAt: time.Now(),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			trip.ID,
		)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, trip)
	})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	var status string

	err = pool.QueryRow(
		ctx,
		"SELECT status FROM trips WHERE id = $1",
		trip.ID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("get created trip: %v", err)
	}

	if status != string(domaintrip.StatusActive) {
		t.Errorf("status = %q, want %q", status, domaintrip.StatusActive)
	}

	var historyStatus string
	var reason string

	err = pool.QueryRow(
		ctx,
		`SELECT to_status, reason
		 FROM trip_status_history
		 WHERE trip_id = $1`,
		trip.ID,
	).Scan(&historyStatus, &reason)
	if err != nil {
		t.Fatalf("get trip history: %v", err)
	}

	if historyStatus != string(domaintrip.StatusActive) {
		t.Errorf("history status = %q, want %q", historyStatus, domaintrip.StatusActive)
	}

	if reason != "created" {
		t.Errorf("history reason = %q, want %q", reason, "created")
	}
}

func TestRepository_Create_DriverBusy_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	driverID := uuid.New()

	firstTrip := model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
		Price:      500,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now(),
	}

	secondTrip := firstTrip
	secondTrip.ID = uuid.New()
	secondTrip.UserID = uuid.New()

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = ANY($1)",
			[]uuid.UUID{firstTrip.ID, secondTrip.ID},
		)
		if err != nil {
			t.Errorf("cleanup trips: %v", err)
		}
	})

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, firstTrip)
	})
	if err != nil {
		t.Fatalf("create first trip: %v", err)
	}

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, secondTrip)
	})

	if !errors.Is(err, domaintrip.ErrDriverBusy) {
		t.Fatalf("got error %v, want ErrDriverBusy", err)
	}
}

func TestRepository_GetByID_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	want := model.Trip{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		StartPoint: model.Coordinates{
			Latitude:  55.75,
			Longitude: 37.62,
		},
		EndPoint: model.Coordinates{
			Latitude:  55.76,
			Longitude: 37.63,
		},
		Price:     500,
		Status:    domaintrip.StatusActive,
		StartedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			want.ID,
		)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, want)
	})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	got, err := repo.GetByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("get trip: %v", err)
	}

	if got.ID != want.ID {
		t.Errorf("ID = %s, want %s", got.ID, want.ID)
	}

	if got.UserID != want.UserID {
		t.Errorf("UserID = %s, want %s", got.UserID, want.UserID)
	}

	if got.DriverID != want.DriverID {
		t.Errorf("DriverID = %s, want %s", got.DriverID, want.DriverID)
	}

	if got.StartPoint != want.StartPoint {
		t.Errorf("StartPoint = %+v, want %+v", got.StartPoint, want.StartPoint)
	}

	if got.EndPoint != want.EndPoint {
		t.Errorf("EndPoint = %+v, want %+v", got.EndPoint, want.EndPoint)
	}

	if got.Price != want.Price {
		t.Errorf("Price = %d, want %d", got.Price, want.Price)
	}

	if got.Status != want.Status {
		t.Errorf("Status = %q, want %q", got.Status, want.Status)
	}

	if !got.StartedAt.Equal(want.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, want.StartedAt)
	}

	if got.FinishedAt != nil {
		t.Errorf("FinishedAt = %v, want nil", got.FinishedAt)
	}
}

func TestRepository_GetByID_NotFound_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)

	_, err = repo.GetByID(ctx, uuid.New())

	if !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Fatalf("got error %v, want ErrTripNotFound", err)
	}
}

func TestRepository_Finish_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	trip := model.Trip{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		StartPoint: model.Coordinates{
			Latitude:  55.75,
			Longitude: 37.62,
		},
		EndPoint: model.Coordinates{
			Latitude:  55.76,
			Longitude: 37.63,
		},
		Price:     500,
		Status:    domaintrip.StatusActive,
		StartedAt: time.Now(),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			trip.ID,
		)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, trip)
	})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	trip.Status = domaintrip.StatusCompleted
	trip.FinishedAt = &finishedAt

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Finish(ctx, trip.ID, trip)
	})
	if err != nil {
		t.Fatalf("finish trip: %v", err)
	}

	var status string
	var savedFinishedAt time.Time

	err = pool.QueryRow(
		ctx,
		"SELECT status, finished_at FROM trips WHERE id = $1",
		trip.ID,
	).Scan(&status, &savedFinishedAt)
	if err != nil {
		t.Fatalf("get finished trip: %v", err)
	}

	if status != string(domaintrip.StatusCompleted) {
		t.Errorf("status = %q, want %q", status, domaintrip.StatusCompleted)
	}

	if !savedFinishedAt.Equal(finishedAt) {
		t.Errorf("finished_at = %v, want %v", savedFinishedAt, finishedAt)
	}

	var historyStatus string
	var reason string

	err = pool.QueryRow(
		ctx,
		`SELECT to_status, reason
		 FROM trip_status_history
		 WHERE trip_id = $1 AND to_status = 'completed'`,
		trip.ID,
	).Scan(&historyStatus, &reason)
	if err != nil {
		t.Fatalf("get finish history: %v", err)
	}

	if historyStatus != string(domaintrip.StatusCompleted) {
		t.Errorf("history status = %q, want %q", historyStatus, domaintrip.StatusCompleted)
	}

	if reason != "finished by user" {
		t.Errorf("history reason = %q, want %q", reason, "finished by user")
	}
}

func TestRepository_Finish_AlreadyCompleted_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
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
		StartedAt:  time.Now(),
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			trip.ID,
		)
		if err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Create(ctx, trip)
	})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	trip.Status = domaintrip.StatusCompleted
	trip.FinishedAt = &finishedAt

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Finish(ctx, trip.ID, trip)
	})
	if err != nil {
		t.Fatalf("first finish: %v", err)
	}

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Finish(ctx, trip.ID, trip)
	})

	if !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Fatalf("got error %v, want ErrTripCompleted", err)
	}
}

func TestRepository_Finish_NotFound_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repo := New(pool)
	txm := txmanager.New(pool)

	finishedAt := time.Now().UTC().Truncate(time.Microsecond)

	trip := model.Trip{
		ID:         uuid.New(),
		Status:     domaintrip.StatusCompleted,
		FinishedAt: &finishedAt,
	}

	err = txm.Do(ctx, func(ctx context.Context) error {
		return repo.Finish(ctx, trip.ID, trip)
	})

	if !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Fatalf("got error %v, want ErrTripNotFound", err)
	}
}
