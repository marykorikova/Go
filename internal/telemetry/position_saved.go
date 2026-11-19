package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type PositionSavedMetrics struct {
	saved metric.Int64Counter
}

func NewPositionSavedMetrics() (*PositionSavedMetrics, error) {
	counter, err := otel.Meter("trip-service").Int64Counter(
		"tripgo.positions.saved",
		metric.WithDescription("Number of successfully saved trip positions"),
	)
	if err != nil {
		return nil, err
	}

	return &PositionSavedMetrics{saved: counter}, nil
}

func (m *PositionSavedMetrics) RecordSaved(ctx context.Context) {
	m.saved.Add(ctx, 1)
}
