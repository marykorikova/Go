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

func TestListStale_Integration(t *testing.T) {
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
	now := time.Now().UTC().Truncate(time.Microsecond)
	staleBefore := now.Add(-60 * time.Second)

	type testCase struct {
		name           string
		status         domaintrip.Status
		startedAt      time.Time
		lastPositionAt *time.Time
	}

	oldPosition := now.Add(-2 * time.Minute)
	freshPosition := now.Add(-10 * time.Second)

	cases := []testCase{
		{
			name:      "without_positions",
			status:    domaintrip.StatusActive,
			startedAt: now.Add(-3 * time.Minute),
		},
		{
			name:           "stale_position",
			status:         domaintrip.StatusActive,
			startedAt:      now.Add(-5 * time.Minute),
			lastPositionAt: &oldPosition,
		},
		{
			name:           "fresh_position",
			status:         domaintrip.StatusActive,
			startedAt:      now.Add(-5 * time.Minute),
			lastPositionAt: &freshPosition,
		},
		{
			name:           "completed",
			status:         domaintrip.StatusCompleted,
			startedAt:      now.Add(-5 * time.Minute),
			lastPositionAt: &oldPosition,
		},
	}

	tripIDs := make(map[string]uuid.UUID)

	t.Cleanup(func() {
		ids := make([]uuid.UUID, 0, len(tripIDs))
		for _, id := range tripIDs {
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return
		}

		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = ANY($1)",
			ids,
		)
		if err != nil {
			t.Errorf("cleanup trips: %v", err)
		}
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tripID := uuid.New()
			tripIDs[tc.name] = tripID

			trip := model.Trip{
				ID:         tripID,
				UserID:     uuid.New(),
				DriverID:   uuid.New(),
				StartPoint: model.Coordinates{Latitude: 55.75, Longitude: 37.62},
				EndPoint:   model.Coordinates{Latitude: 55.76, Longitude: 37.63},
				Price:      500,
				Status:     tc.status,
				StartedAt:  tc.startedAt,
			}

			if tc.status == domaintrip.StatusCompleted {
				trip.FinishedAt = &now
			}

			if tc.status == domaintrip.StatusCompleted {
				_, err := pool.Exec(ctx, `
					INSERT INTO trips (
						id, user_id, driver_id,
						start_latitude, start_longitude,
						end_latitude, end_longitude,
						price, status, started_at, finished_at
					) VALUES ($1, $2, $3, 55.75, 37.62, 55.76, 37.63,
						500, 'completed', $4, $5)
				`, trip.ID, trip.UserID, trip.DriverID, trip.StartedAt, now)
				if err != nil {
					t.Fatalf("create completed test trip: %v", err)
				}
				return
			}

			if err := txm.Do(ctx, func(txCtx context.Context) error {
				if err := repo.Create(txCtx, trip); err != nil {
					return err
				}

				if tc.lastPositionAt != nil {
					if err := repo.LockForPosition(txCtx, tripID); err != nil {
						return err
					}
					_, err := repo.SavePosition(
						txCtx,
						tripID,
						model.Coordinates{Latitude: 59.935, Longitude: 30.335},
						*tc.lastPositionAt,
					)
					return err
				}
				return nil
			}); err != nil {
				t.Fatalf("create test trip: %v", err)
			}
		})
	}

	staleTrips, err := repo.ListStale(ctx, staleBefore)
	if err != nil {
		t.Fatalf("list stale trips: %v", err)
	}

	if len(staleTrips) != 2 {
		t.Fatalf("stale trips count = %d, want 2", len(staleTrips))
	}

	expectedIDs := []uuid.UUID{
		tripIDs["without_positions"],
		tripIDs["stale_position"],
	}

	for i, expectedID := range expectedIDs {
		if staleTrips[i].TripID != expectedID {
			t.Errorf(
				"staleTrips[%d].TripID = %v, want %v",
				i,
				staleTrips[i].TripID,
				expectedID,
			)
		}
		if staleTrips[i].DriverID == uuid.Nil {
			t.Errorf("staleTrips[%d].DriverID is empty", i)
		}
	}
}
