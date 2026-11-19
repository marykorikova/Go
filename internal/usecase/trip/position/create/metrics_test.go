package create

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
	"github.com/marykorikova/Go/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPositionSavedMetric(t *testing.T) {
	ctx := context.Background()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(metric.NewMeterProvider())
		_ = provider.Shutdown(ctx)
	})

	metrics, err := telemetry.NewPositionSavedMetrics()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		repo *fakeTripRepository
		txm  *fakeTxManager
		want bool
	}{
		{"success", &fakeTripRepository{}, &fakeTxManager{}, true},
		{"transaction error", &fakeTripRepository{}, &fakeTxManager{err: errors.New("transaction failed")}, false},
		{"save error", &fakeTripRepository{saveErr: errors.New("save failed")}, &fakeTxManager{}, false},
	}

	for _, tt := range tests {
		uc := New(tt.repo, tt.txm)
		uc.SetMetrics(metrics)

		_, err := uc.Execute(ctx, uuid.New(), model.Coordinates{}, time.Now())
		if (err == nil) != tt.want {
			t.Fatalf("%s: Execute() error = %v", tt.name, err)
		}
	}

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "tripgo.positions.saved" {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("unexpected metric type: %T", m.Data)
			}
			if len(sum.DataPoints) != 1 {
				t.Fatalf("data points = %d, want 1", len(sum.DataPoints))
			}
			if got := sum.DataPoints[0].Value; got != 1 {
				t.Fatalf("saved positions = %d, want 1", got)
			}
			if attrs := sum.DataPoints[0].Attributes.Len(); attrs != 0 {
				t.Fatalf("attributes = %d, want 0", attrs)
			}
			return
		}
	}

	t.Fatal("tripgo.positions.saved metric not found")
}
