package positionscanner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestScannerDoesNotOverlap(t *testing.T) {
	var running atomic.Int32
	var calls atomic.Int32
	var overlap atomic.Bool

	started := make(chan struct{})
	release := make(chan struct{})

	scanner := New(func(ctx context.Context) error {
		if running.Add(1) != 1 {
			overlap.Store(true)
		}
		defer running.Add(-1)

		calls.Add(1)
		close(started)

		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil
	}, 20*time.Millisecond)

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
		t.Fatal("scanner did not start")
	}

	time.Sleep(50 * time.Millisecond)

	if overlap.Load() {
		t.Error("scanner started overlapping scans")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("expected one scan, got %d", got)
	}

	cancel()
	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner did not stop")
	}
}

func TestScannerContinuesAfterError(t *testing.T) {
	var calls atomic.Int32

	scanner := New(func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("database unavailable")
		}
		return nil
	}, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		scanner.Run(ctx)
		close(done)
	}()

	deadline := time.After(time.Second)
	for calls.Load() < 2 {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("scanner did not continue after error")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	cancel()
	<-done

	if got := calls.Load(); got < 2 {
		t.Errorf("scan calls = %d, want at least 2", got)
	}
}
