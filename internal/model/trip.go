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
