package trip

import (
	"time"

	"github.com/google/uuid"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	id             uuid.UUID
	userId         uuid.UUID
	driverId       uuid.UUID
	price          int64
	status         Status
	startPoint     Point
	endPoint       Point
	startedAt      *time.Time
	finishedAt     *time.Time
	lastPositionAt *time.Time
}

func CreateTrip(id, userId, driverId uuid.UUID, price int64, startPoint, endPoint Point, at time.Time) *Trip {
	return &Trip{
		id:             id,
		userId:         userId,
		driverId:       driverId,
		price:          price,
		status:         Active,
		startPoint:     startPoint,
		endPoint:       endPoint,
		startedAt:      &at,
		lastPositionAt: &at,
	}
}

func (t *Trip) Id() uuid.UUID {
	return t.id
}

func (t *Trip) UserId() uuid.UUID {
	return t.userId
}

func (t *Trip) DriverId() uuid.UUID {
	return t.driverId
}

func (t *Trip) Price() int64 {
	return t.price
}

func (t *Trip) Status() Status {
	return t.status
}

func (t *Trip) StartPoint() Point {
	return t.startPoint
}

func (t *Trip) EndPoint() Point {
	return t.endPoint
}

func (t *Trip) StartedAt() *time.Time {
	return t.startedAt
}

func (t *Trip) FinishedAt() *time.Time {
	return t.finishedAt
}

func (t *Trip) LastPositionAt() *time.Time {
	return t.lastPositionAt
}

func (t *Trip) Finish(at time.Time) error {
	if t.status != Active {
		return ErrTripCompleted
	}
	t.status = Completed
	t.finishedAt = &at
	return nil
}

func (t *Trip) SameRequest(other *Trip) bool {
	return t.userId == other.userId &&
		t.driverId == other.driverId &&
		t.startPoint == other.startPoint &&
		t.endPoint == other.endPoint &&
		t.price == other.price
}

func RestoreTrip(
	id uuid.UUID,
	userID uuid.UUID,
	driverID uuid.UUID,
	price int64,
	status Status,
	startPoint Point,
	endPoint Point,
	startedAt *time.Time,
	finishedAt *time.Time,
	lastPositionAt *time.Time,
) *Trip {
	return &Trip{
		id:             id,
		userId:         userID,
		driverId:       driverID,
		price:          price,
		status:         status,
		startPoint:     startPoint,
		endPoint:       endPoint,
		startedAt:      startedAt,
		finishedAt:     finishedAt,
		lastPositionAt: lastPositionAt,
	}
}
