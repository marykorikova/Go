package trip

import (
	"errors"
	"testing"
	"time"
)

func TestTripFinish(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		status     Status
		wantErr    error
		wantStatus Status
	}{
		{
			name:       "active trip can be finished",
			status:     StatusActive,
			wantStatus: StatusCompleted,
		},
		{
			name:       "completed trip cannot be finished",
			status:     StatusCompleted,
			wantErr:    ErrTripCompleted,
			wantStatus: StatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := Trip{Status: tt.status}

			err := trip.Finish(now)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Finish() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if trip.Status != tt.wantStatus {
				t.Errorf(
					"status = %s, want %s",
					trip.Status,
					tt.wantStatus,
				)
			}

			if tt.wantErr == nil {
				if trip.FinishedAt == nil {
					t.Fatal("FinishedAt must be set")
				}
				if !trip.FinishedAt.Equal(now) {
					t.Errorf("unexpected finish time")
				}
			} else if trip.FinishedAt != nil {
				t.Error("failed Finish must not set FinishedAt")
			}
		})
	}
}
