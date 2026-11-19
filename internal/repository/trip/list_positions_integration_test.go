//go:build integration

package trip

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/txmanager"
)

func TestListPositions_Integration(t *testing.T) {
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

	baseTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	inputs := []struct {
		latitude   float64
		recordedAt time.Time
	}{
		{latitude: 30, recordedAt: baseTime.Add(2 * time.Minute)},
		{latitude: 10, recordedAt: baseTime},
		{latitude: 20, recordedAt: baseTime},
	}

	for _, input := range inputs {
		err := txm.Do(ctx, func(txCtx context.Context) error {
			if err := repo.LockForPosition(txCtx, trip.ID); err != nil {
				return err
			}

			_, err := repo.SavePosition(
				txCtx,
				trip.ID,
				model.Coordinates{Latitude: input.latitude, Longitude: 30.335},
				input.recordedAt,
			)
			return err
		})
		if err != nil {
			t.Fatalf("save position: %v", err)
		}
	}

	positions, err := repo.ListPositions(ctx, trip.ID)
	if err != nil {
		t.Fatalf("list positions: %v", err)
	}

	if len(positions) != 3 {
		t.Fatalf("positions count = %d, want 3", len(positions))
	}

	expectedLatitudes := []float64{10, 20, 30}

	for i, expected := range expectedLatitudes {
		if positions[i].Coordinates.Latitude != expected {
			t.Errorf(
				"positions[%d].latitude = %v, want %v",
				i,
				positions[i].Coordinates.Latitude,
				expected,
			)
		}

		if positions[i].TripID != trip.ID {
			t.Errorf("positions[%d].trip_id = %v, want %v",
				i, positions[i].TripID, trip.ID)
		}
	}

	if positions[0].ID >= positions[1].ID {
		t.Errorf(
			"positions with equal recorded_at are not ordered by id: %d >= %d",
			positions[0].ID,
			positions[1].ID,
		)
	}
}

func TestListPositions_TripNotFound_Integration(t *testing.T) {
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
	positions, err := repo.ListPositions(ctx, uuid.New())

	if err != domaintrip.ErrTripNotFound {
		t.Fatalf("error = %v, want ErrTripNotFound", err)
	}
	if len(positions) != 0 {
		t.Errorf("positions count = %d, want 0", len(positions))
	}
}
