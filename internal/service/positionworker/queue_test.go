package positionworker

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestQueueRejectsDuplicate(t *testing.T) {
	q := NewQueue()
	job := Job{TripID: uuid.New(), DriverID: uuid.New()}

	if got := q.Enqueue(job); got != Enqueued {
		t.Fatalf("first enqueue: got %q, want %q", got, Enqueued)
	}

	if got := q.Enqueue(job); got != Duplicate {
		t.Fatalf("duplicate enqueue: got %q, want %q", got, Duplicate)
	}

	if got := len(q.jobs); got != 1 {
		t.Fatalf("queue length: got %d, want 1", got)
	}
}

func TestQueueDropsWhenFull(t *testing.T) {
	q := NewQueue()

	for i := 0; i < QueueCapacity; i++ {
		job := Job{TripID: uuid.New(), DriverID: uuid.New()}

		if got := q.Enqueue(job); got != Enqueued {
			t.Fatalf("enqueue %d: got %q, want %q", i, got, Enqueued)
		}
	}

	extra := Job{TripID: uuid.New(), DriverID: uuid.New()}

	if got := q.Enqueue(extra); got != Dropped {
		t.Fatalf("full queue: got %q, want %q", got, Dropped)
	}

	if got := len(q.jobs); got != QueueCapacity {
		t.Fatalf("queue length: got %d, want %d", got, QueueCapacity)
	}
}

// TestQueueCancel проверяет отмену задания и очистку отметок.
func TestQueueCancel(t *testing.T) {
	queue := NewQueue()
	job := Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}

	if got := queue.Enqueue(job); got != Enqueued {
		t.Fatalf("enqueue: got %q, want %q", got, Enqueued)
	}

	queue.Cancel(job.TripID)

	if !queue.IsCanceled(job.TripID) {
		t.Fatal("expected job to be canceled")
	}

	queue.Complete(job.TripID)

	if queue.IsCanceled(job.TripID) {
		t.Fatal("cancellation flag was not cleared")
	}

	if got := queue.Enqueue(job); got != Enqueued {
		t.Fatalf("enqueue after completion: got %q, want %q", got, Enqueued)
	}
}

func TestQueueCloseDrainsJobs(t *testing.T) {
	q := NewQueue()
	job := Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}

	if got := q.Enqueue(job); got != Enqueued {
		t.Fatalf("Enqueue() = %q, want %q", got, Enqueued)
	}

	q.Close()
	q.Close()

	if got := q.Enqueue(Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}); got != Dropped {
		t.Errorf("Enqueue() after Close() = %q, want %q", got, Dropped)
	}

	got, ok := q.Next(context.Background())
	if !ok || got.TripID != job.TripID {
		t.Fatalf("Next() = (%v, %v), want queued job", got, ok)
	}

	q.Complete(got.TripID)

	if _, ok := q.Next(context.Background()); ok {
		t.Fatal("Next() returned a job after queue was drained")
	}
}

func TestQueueSize(t *testing.T) {
	queue := NewQueue()

	if got := queue.Size(); got != 0 {
		t.Fatalf("empty queue size = %d, want 0", got)
	}

	job := Job{TripID: uuid.New(), DriverID: uuid.New()}
	if result := queue.Enqueue(job); result != Enqueued {
		t.Fatalf("enqueue = %q, want %q", result, Enqueued)
	}

	if got := queue.Size(); got != 1 {
		t.Fatalf("queue size after enqueue = %d, want 1", got)
	}

	if _, ok := queue.Next(context.Background()); !ok {
		t.Fatal("expected queued job")
	}

	if got := queue.Size(); got != 0 {
		t.Fatalf("queue size after dequeue = %d, want 0", got)
	}

	queue.Complete(job.TripID)
}
