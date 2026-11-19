package positionworker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPoolRecordsPushErrorInSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer func() {
		otel.SetTracerProvider(previousProvider)
		_ = provider.Shutdown(context.Background())
	}()

	queue := NewQueue()
	processed := make(chan struct{}, 1)
	pushErr := errors.New("push service unavailable")

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		processed <- struct{}{}
		return pushErr
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	if got := queue.Enqueue(Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}); got != Enqueued {
		t.Fatalf("enqueue: got %q", got)
	}

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("worker did not process job")
	}

	queue.Close()
	pool.Wait()

	for _, span := range recorder.Ended() {
		if span.Name() != "position.worker" {
			continue
		}

		if span.Status().Code != codes.Error {
			t.Fatalf("span status = %v, want Error", span.Status().Code)
		}
		if len(span.Events()) == 0 {
			t.Fatal("span has no recorded error event")
		}
		return
	}

	t.Fatal("position.worker span not found")
}
