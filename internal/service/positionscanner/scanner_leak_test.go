package positionscanner

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestScannerStopsWithoutGoroutineLeaks(t *testing.T) {
	defer goleak.VerifyNone(t)

	started := make(chan struct{})
	scanner := New(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}, time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		scanner.Run(ctx)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("scanner did not start")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scanner did not stop")
	}
}
