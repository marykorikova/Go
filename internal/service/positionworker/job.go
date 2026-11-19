package positionworker

import (
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// Job — задание на отправку водителю запроса координат.
type Job struct {
	TripID      uuid.UUID
	DriverID    uuid.UUID
	SpanContext trace.SpanContext
}

const (
	WorkerCount   = 4
	QueueCapacity = 100
)
