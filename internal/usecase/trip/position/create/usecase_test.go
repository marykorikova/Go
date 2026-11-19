package create

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marykorikova/Go/internal/model"
)

type fakePositionJobCanceler struct {
	called bool
	tripID uuid.UUID
}

func (f *fakePositionJobCanceler) Cancel(tripID uuid.UUID) {
	f.called = true
	f.tripID = tripID
}

type fakeTxManager struct {
	err error
}

func (f *fakeTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

type fakeTripRepository struct {
	saveErr error
}

func (f *fakeTripRepository) LockForPosition(ctx context.Context, tripID uuid.UUID) error {
	return nil
}

func (f *fakeTripRepository) SavePosition(
	ctx context.Context,
	tripID uuid.UUID,
	coordinates model.Coordinates,
	recordedAt time.Time,
) (model.TripPosition, error) {
	if f.saveErr != nil {
		return model.TripPosition{}, f.saveErr
	}

	return model.TripPosition{
		TripID:      tripID,
		Coordinates: coordinates,
		RecordedAt:  recordedAt,
	}, nil
}

func TestExecuteCancelsJobAfterSuccessfulTransaction(t *testing.T) {
	tripID := uuid.New()
	canceler := &fakePositionJobCanceler{}

	uc := New(
		&fakeTripRepository{},
		&fakeTxManager{},
		canceler,
	)

	_, err := uc.Execute(
		context.Background(),
		tripID,
		model.Coordinates{},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !canceler.called {
		t.Fatal("Cancel() was not called")
	}

	if canceler.tripID != tripID {
		t.Errorf("Cancel() tripID = %s, want %s", canceler.tripID, tripID)
	}
}

func TestExecuteDoesNotCancelJobOnTransactionError(t *testing.T) {
	canceler := &fakePositionJobCanceler{}

	uc := New(
		&fakeTripRepository{},
		&fakeTxManager{err: errors.New("transaction failed")},
		canceler,
	)

	_, err := uc.Execute(
		context.Background(),
		uuid.New(),
		model.Coordinates{},
		time.Now(),
	)

	if err == nil {
		t.Fatal("Execute() expected an error")
	}

	if canceler.called {
		t.Fatal("Cancel() must not be called after transaction error")
	}
}

func TestExecuteDoesNotCancelJobOnSaveError(t *testing.T) {
	canceler := &fakePositionJobCanceler{}

	uc := New(
		&fakeTripRepository{saveErr: errors.New("save failed")},
		&fakeTxManager{},
		canceler,
	)

	_, err := uc.Execute(
		context.Background(),
		uuid.New(),
		model.Coordinates{},
		time.Now(),
	)

	if err == nil {
		t.Fatal("Execute() expected a save error")
	}

	if canceler.called {
		t.Fatal("Cancel() must not be called after save error")
	}
}
