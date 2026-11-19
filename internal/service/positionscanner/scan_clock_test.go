package positionscanner

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	triprepo "github.com/marykorikova/Go/internal/repository/trip"
	"github.com/marykorikova/Go/internal/service/positionworker"
)

type clockRepositoryStub struct {
	staleBefore time.Time
}

func (r *clockRepositoryStub) ListStale(
	ctx context.Context,
	staleBefore time.Time,
) ([]triprepo.StaleTrip, error) {
	r.staleBefore = staleBefore
	return nil, nil
}

func TestDatabaseScanUsesInjectedClock(t *testing.T) {
	fixedTime := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	repo := &clockRepositoryStub{}
	queue := positionworker.NewQueue()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	scan := NewDatabaseScanWithClock(
		repo,
		queue,
		5*time.Minute,
		log,
		func() time.Time { return fixedTime },
	)

	if err := scan(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := fixedTime.Add(-5 * time.Minute)
	if !repo.staleBefore.Equal(want) {
		t.Fatalf("staleBefore = %s, want %s", repo.staleBefore, want)
	}
}
