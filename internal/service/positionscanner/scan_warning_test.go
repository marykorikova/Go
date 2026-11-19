package positionscanner

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	triprepo "github.com/marykorikova/Go/internal/repository/trip"
	"github.com/marykorikova/Go/internal/service/positionworker"
)

type staleRepositoryStub struct {
	trips []triprepo.StaleTrip
}

func (r staleRepositoryStub) ListStale(
	ctx context.Context,
	staleBefore time.Time,
) ([]triprepo.StaleTrip, error) {
	return r.trips, nil
}

func TestDatabaseScanWarnsWhenQueueFull(t *testing.T) {
	queue := positionworker.NewQueue()

	for i := 0; i < positionworker.QueueCapacity; i++ {
		result := queue.Enqueue(positionworker.Job{
			TripID:   uuid.New(),
			DriverID: uuid.New(),
		})
		if result != positionworker.Enqueued {
			t.Fatalf("fill queue %d: got %q", i, result)
		}
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	repo := staleRepositoryStub{
		trips: []triprepo.StaleTrip{
			{TripID: uuid.New(), DriverID: uuid.New()},
		},
	}

	scan := NewDatabaseScan(repo, queue, time.Minute, logger)
	if err := scan(context.Background()); err != nil {
		t.Fatalf("scan: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, "level=WARN") {
		t.Fatalf("expected WARN log, got:\n%s", output)
	}
	if !strings.Contains(output, "result=dropped") {
		t.Fatalf("expected dropped result, got:\n%s", output)
	}
}

func TestDatabaseScanStopsOnCanceledContext(t *testing.T) {
	queue := positionworker.NewQueue()
	repo := staleRepositoryStub{
		trips: []triprepo.StaleTrip{
			{TripID: uuid.New(), DriverID: uuid.New()},
		},
	}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	scan := NewDatabaseScan(repo, queue, time.Minute, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := scan(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("scan error = %v, want context.Canceled", err)
	}

	if got := queue.Enqueue(positionworker.Job{
		TripID:   repo.trips[0].TripID,
		DriverID: repo.trips[0].DriverID,
	}); got != positionworker.Enqueued {
		t.Fatalf("enqueue after canceled scan = %q, want %q", got, positionworker.Enqueued)
	}
}
