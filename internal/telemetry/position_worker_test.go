package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionWorkerMetrics(t *testing.T) {
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

	metrics.RecordJob(ctx, "success")
	metrics.RecordJob(ctx, "success")
	metrics.RecordJob(ctx, "error")
	metrics.RecordJob(ctx, "dropped")
	metrics.RecordJob(ctx, "dropped")

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}

	counts := make(map[string]int64)
	found := false

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "tripgo.position_requests" {
				continue
			}

			found = true
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

	if !found {
		t.Fatal("tripgo.position_requests not found")
	}

	for result, want := range map[string]int64{
		"success": 2,
		"error":   1,
		"dropped": 2,
	} {
		if got := counts[result]; got != want {
			t.Errorf("%s: got %d, want %d", result, got, want)
		}
	}
}
