package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tripfinish "github.com/marykorikova/Go/internal/http/trip/finish"
	tripget "github.com/marykorikova/Go/internal/http/trip/get"
	"github.com/marykorikova/Go/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHandler_Health(t *testing.T) {
	handler := &Handler{}

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Health(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	wantBody := `{"status":"ok"}`

	if got := recorder.Body.String(); got != wantBody {
		t.Errorf("body = %q, want %q", got, wantBody)
	}
}

func TestHandler_Ready_Unavailable(t *testing.T) {
	ctx := context.Background()

	pool, err := pgxpool.New(
		ctx,
		"postgres://test:test@127.0.0.1:55432/test?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	pool.Close()

	handler := &Handler{
		pool:         pool,
		queryTimeout: time.Second,
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/ready",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.Ready(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusServiceUnavailable,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	if got := recorder.Body.String(); got != `{"status":"unavailable"}` {
		t.Errorf(
			"body = %q, want %q",
			got,
			`{"status":"unavailable"}`,
		)
	}
}

func TestWriteError(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/trips/123",
		nil,
	)
	recorder := httptest.NewRecorder()

	writeError(
		recorder,
		request,
		http.StatusNotFound,
		"trip_not_found",
		"Trip not found",
	)

	if recorder.Code != http.StatusNotFound {
		t.Errorf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusNotFound,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}

	var body map[string]interface{}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["code"] != "trip_not_found" {
		t.Errorf("code = %v, want trip_not_found", body["code"])
	}

	if body["instance"] != "/trips/123" {
		t.Errorf("instance = %v, want /trips/123", body["instance"])
	}

	if body["status"] != float64(http.StatusNotFound) {
		t.Errorf("status in body = %v, want 404", body["status"])
	}
}

func TestHandler_PositionsNotImplemented(t *testing.T) {
	tests := []struct {
		name   string
		method string
		handle func(http.ResponseWriter, *http.Request)
	}{
		{
			name:   "list positions",
			method: http.MethodGet,
			handle: func(w http.ResponseWriter, r *http.Request) {
				(&Handler{}).ListTripPositions(w, r, uuid.New())
			},
		},
		{
			name:   "create position",
			method: http.MethodPost,
			handle: func(w http.ResponseWriter, r *http.Request) {
				(&Handler{}).CreateTripPosition(w, r, uuid.New())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				tt.method,
				"/trips/123/positions",
				nil,
			)
			recorder := httptest.NewRecorder()

			tt.handle(recorder, request)

			if recorder.Code != http.StatusNotImplemented {
				t.Errorf(
					"status = %d, want %d",
					recorder.Code,
					http.StatusNotImplemented,
				)
			}
		})
	}
}

func TestNewHandlerWithUsecases(t *testing.T) {
	timeout := 5 * time.Second

	handler := NewHandlerWithUsecases(
		nil,
		timeout,
		nil,
		nil,
		nil,
	)

	if handler == nil {
		t.Fatal("NewHandlerWithUsecases() returned nil")
	}

	if handler.queryTimeout != timeout {
		t.Errorf(
			"queryTimeout = %v, want %v",
			handler.queryTimeout,
			timeout,
		)
	}
}

type fakeTripGetter struct {
	called bool
	gotID  uuid.UUID
}

func (f *fakeTripGetter) Execute(
	ctx context.Context,
	id uuid.UUID,
) (model.Trip, error) {
	f.called = true
	f.gotID = id

	return model.Trip{}, errors.New("test error")
}

func TestHandler_GetTrip_DelegatesToUsecase(t *testing.T) {
	fake := &fakeTripGetter{}

	handler := NewHandlerWithUsecases(
		nil,
		5*time.Second,
		nil,
		tripget.New(fake),
		nil,
	)

	tripID := uuid.New()

	request := httptest.NewRequest(
		http.MethodGet,
		"/trips/"+tripID.String(),
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.GetTrip(recorder, request, tripID)

	if !fake.called {
		t.Fatal("GetTrip did not call usecase.Execute")
	}

	if fake.gotID != tripID {
		t.Errorf("got ID = %v, want %v", fake.gotID, tripID)
	}

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}
}

type fakeTripFinisher struct {
	called bool
	gotID  uuid.UUID
}

func (f *fakeTripFinisher) Execute(
	ctx context.Context,
	id uuid.UUID,
) (model.Trip, error) {
	f.called = true
	f.gotID = id

	return model.Trip{}, errors.New("test error")
}

func TestHandler_FinishTrip_DelegatesToUsecase(t *testing.T) {
	fake := &fakeTripFinisher{}

	handler := NewHandlerWithUsecases(
		nil,
		5*time.Second,
		nil,
		nil,
		tripfinish.New(fake),
	)

	tripID := uuid.New()

	request := httptest.NewRequest(
		http.MethodPost,
		"/trips/"+tripID.String()+"/finish",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.FinishTrip(recorder, request, tripID)

	if !fake.called {
		t.Fatal("FinishTrip did not call usecase.Execute")
	}

	if fake.gotID != tripID {
		t.Errorf("got ID = %v, want %v", fake.gotID, tripID)
	}

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}
}
