package telemetry

import (
	"context"
	"testing"
)

func TestInitTracing(t *testing.T) {
	shutdown, err := InitTracing(context.Background())
	if err != nil {
		t.Fatalf("InitTracing() error = %v", err)
	}

	if shutdown == nil {
		t.Fatal("InitTracing() returned nil shutdown")
	}

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}
}
