package positionworker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPoolCreatesSpanLink(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	)

	otel.SetTracerProvider(provider)
	defer func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		_ = provider.Shutdown(context.Background())
	}()

	_, sourceSpan := provider.Tracer("test").Start(
		context.Background(), "position.scan",
	)
	sourceContext := sourceSpan.SpanContext()
	sourceSpan.End()

	queue := NewQueue()
	processed := make(chan struct{}, 1)

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		processed <- struct{}{}
		return nil
	})

	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(workerCtx)

	job := Job{
		TripID:      uuid.New(),
		DriverID:    uuid.New(),
		SpanContext: sourceContext,
	}

	if got := queue.Enqueue(job); got != Enqueued {
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

		links := span.Links()
		if len(links) != 1 {
			t.Fatalf("got %d links, want 1", len(links))
		}
		if links[0].SpanContext.TraceID() != sourceContext.TraceID() ||
			links[0].SpanContext.SpanID() != sourceContext.SpanID() {
			t.Fatal("worker span link does not match scanner span")
		}
		return
	}

	t.Fatal("position.worker span not found")
}
