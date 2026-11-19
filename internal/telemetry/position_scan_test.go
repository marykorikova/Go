package telemetry

import (
	"context"
	"math"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionScanDuration(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		_ = provider.Shutdown(ctx)
	})

	metrics, err := NewPositionScanMetrics()
	if err != nil {
		t.Fatal(err)
	}

	metrics.RecordDuration(ctx, 250*time.Millisecond)

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "position_scan_duration_seconds" {
				continue
			}

			histogram, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("unexpected metric type: %T", m.Data)
			}
			if len(histogram.DataPoints) != 1 {
				t.Fatalf("got %d data points, want 1", len(histogram.DataPoints))
			}

			point := histogram.DataPoints[0]
			if point.Count != 1 || math.Abs(point.Sum-0.25) > 0.000001 {
				t.Errorf("got count=%d sum=%f, want count=1 sum=0.25", point.Count, point.Sum)
			}
			return
		}
	}

	t.Fatal("position_scan_duration_seconds not found")
}
