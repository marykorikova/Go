package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNew_DefaultLevel(t *testing.T) {
	log := New("info")

	if log == nil {
		t.Fatal("New() returned nil")
	}

	ctx := context.Background()

	if !log.Enabled(ctx, slog.LevelInfo) {
		t.Error("INFO level should be enabled")
	}

	if log.Enabled(ctx, slog.LevelDebug) {
		t.Error("DEBUG level should be disabled")
	}
}

func TestNew_Levels(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"uppercase", "DEBUG", slog.LevelDebug},
		{"unknown", "unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)

			if log == nil {
				t.Fatal("New() returned nil")
			}

			ctx := context.Background()

			if !log.Enabled(ctx, tt.want) {
				t.Errorf("level %s should be enabled", tt.want)
			}

			if log.Enabled(ctx, tt.want-1) {
				t.Errorf("level below %s should be disabled", tt.want)
			}
		})
	}
}
