package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", server.URL)

	shutdown, err := InitMetrics(context.Background())
	if err != nil {
		t.Fatalf("InitMetrics() error = %v", err)
	}

	if shutdown == nil {
		t.Fatal("InitMetrics() returned nil shutdown")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown metrics: %v", err)
	}
}
