package positionworker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/client/push"
)

type fakePushSender struct {
	request push.Request
	calls   int
}

func (f *fakePushSender) Send(_ context.Context, request push.Request) error {
	f.request = request
	f.calls++
	return nil
}

func TestPushHandler(t *testing.T) {
	sender := &fakePushSender{}
	handler := NewPushHandler(sender)

	job := Job{
		TripID:   uuid.New(),
		DriverID: uuid.New(),
	}

	if err := handler(context.Background(), job); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	if sender.calls != 1 {
		t.Errorf("Send() calls = %d, want 1", sender.calls)
	}
	if sender.request.RecipientID != job.DriverID.String() {
		t.Errorf("recipient_id = %q, want %q",
			sender.request.RecipientID, job.DriverID)
	}
	if sender.request.Kind != push.KindRequestPosition {
		t.Errorf("kind = %q, want REQUEST_POSITION", sender.request.Kind)
	}
	if sender.request.Data.TripID != job.TripID.String() {
		t.Errorf("trip_id = %q, want %q",
			sender.request.Data.TripID, job.TripID)
	}
	if _, err := uuid.Parse(sender.request.RequestID); err != nil {
		t.Errorf("request_id = %q, expected valid UUID: %v",
			sender.request.RequestID, err)
	}
}
