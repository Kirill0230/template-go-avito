package domain

import (
	"time"

	"github.com/google/uuid"
)

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

type Point struct {
	Latitude  float64
	Longitude float64
}

type Status string

const (
	Active    Status = "active"
	Completed Status = "completed"
)

func (t *Trip) SameRequest(other *Trip) bool {
	return t.UserId == other.UserId &&
		t.DriverId == other.DriverId &&
		t.StartPoint == other.StartPoint &&
		t.EndPoint == other.EndPoint &&
		t.Price == other.Price
}
