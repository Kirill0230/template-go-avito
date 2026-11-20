package trip

import (
	"fmt"
	"math"
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

func CreateTrip(id, userID, driverID uuid.UUID, price int64, startPoint, endPoint Point, at time.Time) (*Trip, error) {
	if price < 0 {
		return nil, ErrInvalidPrice
	}

	if err := validatePoint("start point", startPoint); err != nil {
		return nil, err
	}
	if err := validatePoint("end point", endPoint); err != nil {
		return nil, err
	}

	return &Trip{
		id:         id,
		userId:     userID,
		driverId:   driverID,
		price:      price,
		status:     Active,
		startPoint: startPoint,
		endPoint:   endPoint,
		startedAt:  &at,
	}, nil
}

func validatePoint(name string, point Point) error {
	if math.IsNaN(point.Latitude) || point.Latitude < -90 || point.Latitude > 90 {
		return fmt.Errorf("%w: %s latitude must be between -90 and 90", ErrInvalidCoordinates, name)
	}
	if math.IsNaN(point.Longitude) || point.Longitude < -180 || point.Longitude > 180 {
		return fmt.Errorf("%w: %s longitude must be between -180 and 180", ErrInvalidCoordinates, name)
	}

	return nil
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

func (t *Trip) AcceptPosition(point Point, recordedAt, now time.Time) error {
	if t.status != Active {
		return ErrTripCompleted
	}
	if err := validatePoint("position", point); err != nil {
		return err
	}
	if recordedAt.IsZero() || recordedAt.After(now.Add(time.Minute)) {
		return ErrInvalidRecordedAt
	}

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
