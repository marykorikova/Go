package telemetry

import (
	"context"
	"math"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionWorkerDuration(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		_ = provider.Shutdown(ctx)
	})

	metrics, err := NewPositionWorkerMetrics()
	if err != nil {
		t.Fatal(err)
	}

	metrics.RecordDuration(ctx, 250*time.Millisecond, "success")

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "tripgo.position_requests.duration" {
				continue
			}
			if m.Unit != "s" {
				t.Fatalf("unit = %q, want s", m.Unit)
			}

			histogram, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("unexpected metric type: %T", m.Data)
			}
			if len(histogram.DataPoints) != 1 {
				t.Fatalf("data points = %d, want 1", len(histogram.DataPoints))
			}

			point := histogram.DataPoints[0]
			result, ok := point.Attributes.Value(attribute.Key("result"))
			if !ok || result.AsString() != "success" {
				t.Fatalf("result attribute = %v, present = %v", result, ok)
			}
			if point.Count != 1 || math.Abs(point.Sum-0.25) > 0.000001 {
				t.Fatalf("count = %d, sum = %v; want 1 and 0.25", point.Count, point.Sum)
			}
			return
		}
	}

	t.Fatal("tripgo.position_requests.duration not found")
}
