package positionworker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/goleak"
)

func TestPoolStopsWithoutGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t)

	queue := NewQueue()
	started := make(chan struct{}, WorkerCount)

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	for i := 0; i < WorkerCount; i++ {
		if got := queue.Enqueue(Job{
			TripID:   uuid.New(),
			DriverID: uuid.New(),
		}); got != Enqueued {
			cancel()
			t.Fatalf("enqueue %d: got %q", i, got)
		}
	}

	for i := 0; i < WorkerCount; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			cancel()
			t.Fatal("not all workers started")
		}
	}

	cancel()
	pool.Wait()
}
