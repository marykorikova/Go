package positionworker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/marykorikova/Go/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler func(context.Context, Job) error

type Pool struct {
	queue   *Queue
	handler Handler
	metrics *telemetry.PositionWorkerMetrics
	wg      sync.WaitGroup
	active  atomic.Int64
}

func NewPool(
	queue *Queue,
	handler Handler,
	metrics ...*telemetry.PositionWorkerMetrics,
) *Pool {
	pool := &Pool{
		queue:   queue,
		handler: handler,
	}
	if len(metrics) > 0 {
		pool.metrics = metrics[0]
	}
	return pool
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < WorkerCount; i++ {
		p.wg.Add(1)

		go func() {
			defer p.wg.Done()

			for {
				job, ok := p.queue.Next(ctx)
				if !ok {
					return
				}
				func() {
					defer p.queue.Complete(job.TripID)
					jobCtx, ok := p.queue.StartJob(ctx, job.TripID)
					if !ok {
						if p.metrics != nil {
							p.metrics.RecordJob(ctx, "dropped")
						}
						return
					}
					options := []trace.SpanStartOption{}
					if job.SpanContext.IsValid() {
						options = append(options, trace.WithLinks(
							trace.Link{SpanContext: job.SpanContext},
						))
					}

					jobCtx, span := otel.Tracer("trip-service").Start(
						jobCtx, "position.worker", options...,
					)
					started := time.Now()
					p.active.Add(1)
					err := func() error {
						defer p.active.Add(-1)
						return p.handler(jobCtx, job)
					}()
					duration := time.Since(started)
					if err != nil && jobCtx.Err() == nil {
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
					}
					span.End()

					result := "success"
					if errors.Is(err, context.Canceled) || jobCtx.Err() != nil {
						result = "canceled"
					} else if err != nil {
						result = "error"
					}

					if p.metrics != nil {
						metricResult := result
						if metricResult == "canceled" {
							metricResult = "dropped"
						}
						p.metrics.RecordJob(ctx, metricResult)
						p.metrics.RecordDuration(ctx, duration, metricResult)
					}

					if err != nil {
						if result == "canceled" {
							slog.Debug("position push canceled", "trip_id", job.TripID)
						} else {
							slog.Error(
								"position push failed",
								"trip_id", job.TripID,
								"error", err,
							)
						}
					}
				}()
			}
		}()
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

// Active возвращает количество воркеров, выполняющих задания.
func (p *Pool) Active() int64 {
	return p.active.Load()
}
