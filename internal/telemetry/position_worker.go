package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionWorkerMetrics struct {
	jobsTotal metric.Int64Counter
	duration  metric.Float64Histogram
}

func NewPositionWorkerMetrics() (*PositionWorkerMetrics, error) {
	counter, err := otel.Meter("trip-service").Int64Counter(
		"tripgo.position_requests",
		metric.WithDescription("Number of position jobs by processing result"),
	)
	if err != nil {
		return nil, err
	}

	histogram, err := otel.Meter("trip-service").Float64Histogram(
		"tripgo.position_requests.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of position requests to Push Service"),
	)
	if err != nil {
		return nil, err
	}

	return &PositionWorkerMetrics{
		jobsTotal: counter,
		duration:  histogram,
	}, nil
}

func (m *PositionWorkerMetrics) RecordJob(ctx context.Context, result string) {
	m.jobsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(attribute.String("result", result)),
	)
}

func (m *PositionWorkerMetrics) RecordDuration(ctx context.Context, duration time.Duration, result string) {
	m.duration.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(attribute.String("result", result)),
	)
}

// RegisterActiveWorkers регистрирует метрику занятых воркеров.
func RegisterActiveWorkers(active func() int64) (metric.Registration, error) {
	meter := otel.Meter("trip-service")

	gauge, err := meter.Int64ObservableGauge(
		"tripgo.position_requests.workers.active",
		metric.WithDescription("Number of workers processing position requests"),
		metric.WithUnit("{worker}"),
	)
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			observer.ObserveInt64(gauge, active())
			return nil
		},
		gauge,
	)
}
