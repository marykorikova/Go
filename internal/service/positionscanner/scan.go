package positionscanner

import (
	"context"
	"log/slog"
	"time"

	triprepo "github.com/marykorikova/Go/internal/repository/trip"
	"github.com/marykorikova/Go/internal/service/positionworker"
	"github.com/marykorikova/Go/internal/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type StaleTripRepository interface {
	ListStale(ctx context.Context, staleBefore time.Time) ([]triprepo.StaleTrip, error)
}

func NewDatabaseScan(
	repo StaleTripRepository,
	queue *positionworker.Queue,
	staleAfter time.Duration,
	log *slog.Logger,
	metrics ...*telemetry.PositionQueueMetrics,
) func(context.Context) error {
	return NewDatabaseScanWithClock(repo, queue, staleAfter, log, time.Now, metrics...)
}

func NewDatabaseScanWithClock(
	repo StaleTripRepository,
	queue *positionworker.Queue,
	staleAfter time.Duration,
	log *slog.Logger,
	now func() time.Time,
	metrics ...*telemetry.PositionQueueMetrics,
) func(context.Context) error {
	return func(ctx context.Context) error {
		staleBefore := now().Add(-staleAfter)

		trips, err := repo.ListStale(ctx, staleBefore)
		if err != nil {
			return err
		}

		log.Info("stale trips scanned", "count", len(trips))

		for _, trip := range trips {
			if err := ctx.Err(); err != nil {
				return err
			}
			job := positionworker.Job{
				TripID:      trip.TripID,
				DriverID:    trip.DriverID,
				SpanContext: trace.SpanFromContext(ctx).SpanContext(),
			}

			result := queue.Enqueue(job)
			if len(metrics) > 0 && metrics[0] != nil {
				metrics[0].RecordEnqueue(ctx, string(result))
			}
			if result == positionworker.Dropped {
				log.Warn(
					"position queue full or closed",
					"trip_id", trip.TripID,
					"result", result,
				)
			} else {
				log.Info(
					"stale trip enqueue",
					"trip_id", trip.TripID,
					"result", result,
				)
			}
		}

		return nil
	}
}
