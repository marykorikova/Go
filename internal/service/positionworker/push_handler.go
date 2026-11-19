package positionworker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/client/push"
)

type PushSender interface {
	Send(ctx context.Context, notification push.Request) error
}

func NewPushHandler(sender PushSender) Handler {
	return func(ctx context.Context, job Job) error {
		notification := push.Request{
			RequestID:   uuid.NewString(),
			RecipientID: job.DriverID.String(),
			Kind:        push.KindRequestPosition,
			Data: push.RequestData{
				TripID: job.TripID.String(),
			},
		}

		if err := sender.Send(ctx, notification); err != nil {
			return fmt.Errorf("send position request for trip %s: %w", job.TripID, err)
		}

		return nil
	}
}
