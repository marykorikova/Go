package domainerror

import (
	"encoding/json"
	"errors"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
	"net/http"
)

func Write(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	detail string,
) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     "https://tripgo.example/problems/" + code,
		"title":    code,
		"status":   status,
		"detail":   detail,
		"instance": r.URL.Path,
		"code":     code,
	})
}

func WriteDomainError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, domaintrip.ErrTripNotFound):
		Write(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")

	case errors.Is(err, domaintrip.ErrTripCompleted):
		Write(w, r, http.StatusConflict, "trip_completed", "Trip already completed")

	case errors.Is(err, domaintrip.ErrDriverBusy):
		Write(w, r, http.StatusConflict, "driver_busy", "Driver already has an active trip")

	default:
		Write(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
