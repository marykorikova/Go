package positionworker

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type EnqueueResult string

const (
	Enqueued  EnqueueResult = "enqueued"
	Duplicate EnqueueResult = "duplicate"
	Dropped   EnqueueResult = "dropped"
)

type Queue struct {
	done     chan struct{}
	jobs     chan Job
	mu       sync.Mutex
	pending  map[uuid.UUID]struct{}
	canceled map[uuid.UUID]struct{}
	running  map[uuid.UUID]context.CancelFunc
}

func NewQueue() *Queue {
	return &Queue{
		done:     make(chan struct{}),
		jobs:     make(chan Job, QueueCapacity),
		pending:  make(map[uuid.UUID]struct{}),
		canceled: make(map[uuid.UUID]struct{}),
		running:  make(map[uuid.UUID]context.CancelFunc),
	}
}

func (q *Queue) Enqueue(job Job) EnqueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	select {
	case <-q.done:
		return Dropped
	default:
	}

	if _, exists := q.pending[job.TripID]; exists {
		return Duplicate
	}

	select {
	case q.jobs <- job:
		q.pending[job.TripID] = struct{}{}
		return Enqueued
	default:
		return Dropped
	}
}

// Complete освобождает поездку после завершения обработки задания.
func (q *Queue) Complete(tripID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if cancel, running := q.running[tripID]; running {
		cancel()
		delete(q.running, tripID)
	}

	delete(q.pending, tripID)
	delete(q.canceled, tripID)
}

// Cancel помечает ожидающее задание как устаревшее.
func (q *Queue) Cancel(tripID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.pending[tripID]; exists {
		q.canceled[tripID] = struct{}{}

		if cancel, running := q.running[tripID]; running {
			cancel()
		}
	}
}

// IsCanceled проверяет, было ли задание отменено.
func (q *Queue) IsCanceled(tripID uuid.UUID) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	_, canceled := q.canceled[tripID]
	return canceled
}

// StartJob создаёт контекст задания и регистрирует его отмену.
func (q *Queue) StartJob(parent context.Context, tripID uuid.UUID) (context.Context, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, canceled := q.canceled[tripID]; canceled {
		return nil, false
	}

	ctx, cancel := context.WithCancel(parent)
	q.running[tripID] = cancel

	return ctx, true
}

// Close запрещает добавление новых задач в очередь.
// Повторный вызов безопасен.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	select {
	case <-q.done:
		return
	default:
		close(q.done)
	}
}

func (q *Queue) Next(ctx context.Context) (Job, bool) {
	for {
		select {
		case job := <-q.jobs:
			return job, true
		case <-q.done:
			select {
			case job := <-q.jobs:
				return job, true
			default:
				return Job{}, false
			}
		case <-ctx.Done():
			return Job{}, false
		}
	}
}

// Size возвращает количество заданий, ожидающих обработки.
func (q *Queue) Size() int {
	return len(q.jobs)
}
