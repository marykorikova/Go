package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionQueueMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		_ = provider.Shutdown(context.Background())
	})

	metrics, err := NewPositionQueueMetrics()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	metrics.RecordEnqueue(ctx, "enqueued")
	metrics.RecordEnqueue(ctx, "enqueued")
	metrics.RecordEnqueue(ctx, "duplicate")
	metrics.RecordEnqueue(ctx, "dropped")

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}

	counts := make(map[string]int64)

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "tripgo.stale_checks" {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("unexpected metric type: %T", m.Data)
			}

			for _, point := range sum.DataPoints {
				value, ok := point.Attributes.Value(attribute.Key("result"))
				if !ok {
					t.Fatal("missing result attribute")
				}
				counts[value.AsString()] = point.Value
			}
		}
	}

	for result, want := range map[string]int64{
		"queued":    2,
		"duplicate": 1,
		"dropped":   1,
	} {
		if got := counts[result]; got != want {
			t.Errorf("%s: got %d, want %d", result, got, want)
		}
	}
}
