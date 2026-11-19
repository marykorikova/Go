package positionworker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPoolUsesFourWorkers(t *testing.T) {
	queue := NewQueue()

	var active atomic.Int32
	var maximum atomic.Int32
	started := make(chan struct{}, WorkerCount)
	release := make(chan struct{})

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		n := active.Add(1)
		defer active.Add(-1)

		for {
			old := maximum.Load()
			if n <= old || maximum.CompareAndSwap(old, n) {
				break
			}
		}

		started <- struct{}{}

		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	for i := 0; i < WorkerCount+1; i++ {
		if got := queue.Enqueue(Job{
			TripID:   uuid.New(),
			DriverID: uuid.New(),
		}); got != Enqueued {
			t.Fatalf("enqueue %d: %s", i, got)
		}
	}

	for i := 0; i < WorkerCount; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("four workers did not start processing")
		}
	}

	if got := maximum.Load(); got != WorkerCount {
		t.Errorf("maximum concurrent handlers = %d, want %d", got, WorkerCount)
	}

	if got := len(queue.jobs); got != 1 {
		t.Errorf("queued jobs = %d, want 1", got)
	}

	cancel()
	close(release)
	pool.Wait()
}

func TestPoolAllowsEnqueueAfterCompletion(t *testing.T) {
	queue := NewQueue()
	processed := make(chan struct{}, 1)
	release := make(chan struct{})

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		processed <- struct{}{}
		<-release
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool.Start(ctx)

	job := Job{TripID: uuid.New(), DriverID: uuid.New()}

	if got := queue.Enqueue(job); got != Enqueued {
		t.Fatalf("first enqueue: got %q", got)
	}

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	if got := queue.Enqueue(job); got != Duplicate {
		t.Errorf("enqueue during processing: got %q, want %q", got, Duplicate)
	}

	close(release)

	deadline := time.After(time.Second)
	for {
		if got := queue.Enqueue(job); got == Enqueued {
			break
		}
		select {
		case <-deadline:
			t.Fatal("job was not released after processing")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	cancel()
	pool.Wait()
}

func TestPoolSkipsCanceledJob(t *testing.T) {
	queue := NewQueue()
	started := make(chan struct{}, WorkerCount)
	release := make(chan struct{})
	var canceledProcessed atomic.Bool

	canceledJob := Job{TripID: uuid.New(), DriverID: uuid.New()}

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		if job.TripID == canceledJob.TripID {
			canceledProcessed.Store(true)
			return nil
		}

		started <- struct{}{}
		<-release
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	for i := 0; i < WorkerCount; i++ {
		job := Job{TripID: uuid.New(), DriverID: uuid.New()}
		if got := queue.Enqueue(job); got != Enqueued {
			t.Fatalf("enqueue blocking job: got %q", got)
		}
	}

	for i := 0; i < WorkerCount; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start")
		}
	}

	if got := queue.Enqueue(canceledJob); got != Enqueued {
		t.Fatalf("enqueue canceled job: got %q", got)
	}

	queue.Cancel(canceledJob.TripID)
	close(release)

	deadline := time.After(time.Second)
	for {
		queue.mu.Lock()
		_, pending := queue.pending[canceledJob.TripID]
		queue.mu.Unlock()

		if !pending {
			break
		}

		select {
		case <-deadline:
			t.Fatal("canceled job was not removed from pending")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	cancel()
	pool.Wait()

	if canceledProcessed.Load() {
		t.Fatal("worker processed a canceled job")
	}
}

func TestPoolCancelsRunningJob(t *testing.T) {
	queue := NewQueue()
	started := make(chan struct{})
	canceled := make(chan struct{})

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	job := Job{TripID: uuid.New(), DriverID: uuid.New()}
	if got := queue.Enqueue(job); got != Enqueued {
		t.Fatalf("enqueue: got %q, want %q", got, Enqueued)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	queue.Cancel(job.TripID)

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("running job did not receive cancellation")
	}

	cancel()
	pool.Wait()
}

func TestPoolDrainsQueueOnClose(t *testing.T) {
	queue := NewQueue()
	var processed atomic.Int32

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		processed.Add(1)
		return nil
	})

	for i := 0; i < 3; i++ {
		if result := queue.Enqueue(Job{
			TripID:   uuid.New(),
			DriverID: uuid.New(),
		}); result != Enqueued {
			t.Fatalf("Enqueue() = %q, want %q", result, Enqueued)
		}
	}

	pool.Start(context.Background())
	queue.Close()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers did not stop after draining queue")
	}

	if got := processed.Load(); got != 3 {
		t.Errorf("processed jobs = %d, want 3", got)
	}
}

func TestPoolContinuesAfterHandlerError(t *testing.T) {
	queue := NewQueue()
	var calls atomic.Int32

	pool := NewPool(queue, func(ctx context.Context, job Job) error {
		if calls.Add(1) == 1 {
			return fmt.Errorf("push service unavailable")
		}
		return nil
	})

	for i := 0; i < 2; i++ {
		if result := queue.Enqueue(Job{
			TripID:   uuid.New(),
			DriverID: uuid.New(),
		}); result != Enqueued {
			t.Fatalf("Enqueue() = %q, want %q", result, Enqueued)
		}
	}

	pool.Start(context.Background())
	queue.Close()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers did not finish")
	}

	if got := calls.Load(); got != 2 {
		t.Errorf("handler calls = %d, want 2", got)
	}
}
