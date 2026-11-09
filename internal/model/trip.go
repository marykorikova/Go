package model

import (
	"time"

	"github.com/google/uuid"
	domaintrip "github.com/marykorikova/Go/internal/domain/trip"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         domaintrip.Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}
