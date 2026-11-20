package model

import (
	"time"

	"github.com/google/uuid"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	Id             uuid.UUID
	UserId         uuid.UUID
	DriverId       uuid.UUID
	Price          int64
	Status         Status
	StartPoint     Point
	EndPoint       Point
	StartedAt      *time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

type TripPosition struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Point
	RecordedAt  time.Time
}

type StaleTrip struct {
	ID          uuid.UUID
	DriverID    uuid.UUID
	SilentSince time.Time
}

type StaleTripCursor struct {
	SilentSince time.Time
	ID          uuid.UUID
}
