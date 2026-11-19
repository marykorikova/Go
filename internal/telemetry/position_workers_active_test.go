package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionWorkersActive(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		_ = provider.Shutdown(ctx)
	})

	active := int64(2)
	registration, err := RegisterActiveWorkers(func() int64 { return active })
	if err != nil {
		t.Fatal(err)
	}
	defer registration.Unregister()

	check := func(want int64) {
		t.Helper()

		var data metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &data); err != nil {
			t.Fatal(err)
		}

		for _, scope := range data.ScopeMetrics {
			for _, m := range scope.Metrics {
				if m.Name != "tripgo.position_requests.workers.active" {
					continue
				}

				gauge, ok := m.Data.(metricdata.Gauge[int64])
				if !ok {
					t.Fatalf("unexpected metric type: %T", m.Data)
				}
				if len(gauge.DataPoints) != 1 {
					t.Fatalf("data points = %d, want 1", len(gauge.DataPoints))
				}
				if got := gauge.DataPoints[0].Value; got != want {
					t.Fatalf("active workers = %d, want %d", got, want)
				}
				return
			}
		}

		t.Fatal("tripgo.position_requests.workers.active not found")
	}

	check(2)
	active = 0
	check(0)
}
