package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionQueueMetrics struct {
	enqueueTotal metric.Int64Counter
}

func NewPositionQueueMetrics() (*PositionQueueMetrics, error) {
	counter, err := otel.Meter("trip-service").Int64Counter(
		"tripgo.stale_checks",
		metric.WithDescription("Number of position jobs by enqueue result"),
	)
	if err != nil {
		return nil, err
	}

	return &PositionQueueMetrics{enqueueTotal: counter}, nil
}

func (m *PositionQueueMetrics) RecordEnqueue(ctx context.Context, result string) {
	if result == "enqueued" {
		result = "queued"
	}
	m.enqueueTotal.Add(
		ctx,
		1,
		metric.WithAttributes(attribute.String("result", result)),
	)
}

// RegisterQueueSize регистрирует наблюдаемую метрику размера очереди.
func RegisterQueueSize(size func() int) (metric.Registration, error) {
	meter := otel.Meter("trip-service")

	gauge, err := meter.Int64ObservableGauge(
		"tripgo.position_requests.queue.size",
		metric.WithDescription("Number of position jobs waiting in the queue"),
		metric.WithUnit("{job}"),
	)
	if err != nil {
		return nil, err
	}

	return meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			observer.ObserveInt64(gauge, int64(size()))
			return nil
		},
		gauge,
	)
}
