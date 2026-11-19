package positionworker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPoolActive(t *testing.T) {
	queue := NewQueue()
	started := make(chan struct{})
	release := make(chan struct{})

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		close(started)
		<-release
		return nil
	})

	if got := pool.Active(); got != 0 {
		t.Fatalf("active before start = %d, want 0", got)
	}

	pool.Start(context.Background())

	if result := queue.Enqueue(Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}); result != Enqueued {
		t.Fatalf("enqueue = %q, want %q", result, Enqueued)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	if got := pool.Active(); got != 1 {
		t.Fatalf("active during request = %d, want 1", got)
	}

	close(release)
	queue.Close()
	pool.Wait()

	if got := pool.Active(); got != 0 {
		t.Fatalf("active after completion = %d, want 0", got)
	}
}
