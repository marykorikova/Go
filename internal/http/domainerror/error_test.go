package domainerror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/trips/123",
		nil,
	)
	recorder := httptest.NewRecorder()

	Write(
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

	var body struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
		Code     string `json:"code"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Type != "https://tripgo.example/problems/trip_not_found" {
		t.Errorf("unexpected type: %q", body.Type)
	}

	if body.Title != "trip_not_found" {
		t.Errorf("unexpected title: %q", body.Title)
	}

	if body.Status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", body.Status)
	}

	if body.Detail != "Trip not found" {
		t.Errorf("unexpected detail: %q", body.Detail)
	}

	if body.Instance != "/trips/123" {
		t.Errorf("unexpected instance: %q", body.Instance)
	}

	if body.Code != "trip_not_found" {
		t.Errorf("unexpected code: %q", body.Code)
	}
}
