package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendSuccess(t *testing.T) {
	const (
		requestID = "01J9Z6H7K8QW3M4N5P6R7S8T9V"
		driverID  = "11111111-1111-1111-1111-111111111111"
		tripID    = "22222222-2222-2222-2222-222222222222"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/push" {
			t.Errorf("path = %s, want /api/v1/push", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var request Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}

		if request.RequestID != requestID {
			t.Errorf("request_id = %q, want %q", request.RequestID, requestID)
		}
		if request.RecipientID != driverID {
			t.Errorf("recipient_id = %q, want %q", request.RecipientID, driverID)
		}
		if request.Kind != KindRequestPosition {
			t.Errorf("kind = %q, want %q", request.Kind, KindRequestPosition)
		}
		if request.Data.TripID != tripID {
			t.Errorf("trip_id = %q, want %q", request.Data.TripID, tripID)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(server.URL, time.Second)

	err := client.Send(context.Background(), Request{
		RequestID:   requestID,
		RecipientID: driverID,
		Kind:        KindRequestPosition,
		Data:        RequestData{TripID: tripID},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestSendHTTPErrors(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requestCount := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.WriteHeader(status)
			}))
			defer server.Close()

			client := New(server.URL, time.Second)

			err := client.Send(context.Background(), Request{
				RequestID:   "01J9Z6H7K8QW3M4N5P6R7S8T9V",
				RecipientID: "11111111-1111-1111-1111-111111111111",
				Kind:        KindRequestPosition,
				Data: RequestData{
					TripID: "22222222-2222-2222-2222-222222222222",
				},
			})

			if err == nil {
				t.Fatal("Send() expected error, got nil")
			}

			if requestCount != 1 {
				t.Errorf("request count = %d, want 1 (no retries)", requestCount)
			}
		})
	}
}

func TestSendTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(server.URL, 20*time.Millisecond)

	err := client.Send(context.Background(), Request{
		RequestID:   "01J9Z6H7K8QW3M4N5P6R7S8T9V",
		RecipientID: "11111111-1111-1111-1111-111111111111",
		Kind:        KindRequestPosition,
		Data: RequestData{
			TripID: "22222222-2222-2222-2222-222222222222",
		},
	})

	if err == nil {
		t.Fatal("Send() expected timeout error, got nil")
	}
}
