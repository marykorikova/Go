package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionScanMetrics struct {
	scanTotal metric.Int64Counter
	duration  metric.Float64Histogram
}

func NewPositionScanMetrics() (*PositionScanMetrics, error) {
	counter, err := otel.Meter("trip-service").Int64Counter(
		"position_scan_total",
		metric.WithDescription("Number of position scans by result"),
	)
	if err != nil {
		return nil, err
	}

	histogram, err := otel.Meter("trip-service").Float64Histogram(
		"position_scan_duration_seconds",
		metric.WithDescription("Duration of position scans"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &PositionScanMetrics{
		scanTotal: counter,
		duration:  histogram,
	}, nil
}

func (m *PositionScanMetrics) RecordScan(ctx context.Context, result string) {
	m.scanTotal.Add(
		ctx,
		1,
		metric.WithAttributes(attribute.String("result", result)),
	)
}

func (m *PositionScanMetrics) RecordDuration(ctx context.Context, duration time.Duration) {
	m.duration.Record(ctx, duration.Seconds())
}
