package positionscanner

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/marykorikova/Go/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Scanner struct {
	scan     func(context.Context) error
	interval time.Duration
	metrics  *telemetry.PositionScanMetrics
}

func New(
	scan func(context.Context) error,
	interval time.Duration,
	metrics ...*telemetry.PositionScanMetrics,
) *Scanner {
	scanner := &Scanner{scan: scan, interval: interval}
	if len(metrics) > 0 {
		scanner.metrics = metrics[0]
	}
	return scanner
}

func (s *Scanner) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			started := time.Now()
			scanCtx, span := otel.Tracer("trip-service").Start(ctx, "position.scan")
			err := s.scan(scanCtx)
			if err != nil && ctx.Err() == nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			span.End()
			if s.metrics != nil {
				s.metrics.RecordDuration(ctx, time.Since(started))
			}
			if err != nil {
				if errors.Is(err, context.Canceled) && ctx.Err() != nil {
					return
				}
				if s.metrics != nil {
					s.metrics.RecordScan(ctx, "error")
				}
				slog.Error("position scan failed", "error", err)
			} else if s.metrics != nil {
				s.metrics.RecordScan(ctx, "success")
			}
		}
	}
}
