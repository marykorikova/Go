package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogging(t *testing.T) {
	var output bytes.Buffer

	log := slog.New(slog.NewJSONHandler(&output, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Logging(log)(next)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	var entry struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Status int    `json:"status"`
	}

	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}

	if entry.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", entry.Method)
	}

	if entry.Path != "/api/v1/trips" {
		t.Errorf("path = %q, want /api/v1/trips", entry.Path)
	}

	if entry.Status != http.StatusCreated {
		t.Errorf("logged status = %d, want 201", entry.Status)
	}
}
