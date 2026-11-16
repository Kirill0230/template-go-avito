package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
)

func mustCreateTrip(
	t *testing.T,
	id, userID, driverID uuid.UUID,
	price int64,
	startPoint, endPoint domaintrip.Point,
	at time.Time,
) *domaintrip.Trip {
	t.Helper()

	trip, err := domaintrip.CreateTrip(id, userID, driverID, price, startPoint, endPoint, at)
	if err != nil {
		t.Fatalf("CreateTrip() error: %v", err)
	}

	return trip
}

func TestTripFinish(t *testing.T) {
	startedAt := time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)

	t.Run("active trip", func(t *testing.T) {
		trip := mustCreateTrip(
			t,
			uuid.New(),
			uuid.New(),
			uuid.New(),
			1000,
			domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
			domaintrip.Point{Latitude: 59.93, Longitude: 30.31},
			startedAt,
		)

		err := trip.Finish(finishedAt)
		if err != nil {
			t.Fatalf("Finish() error: %v", err)
		}

		if trip.Status() != domaintrip.Completed {
			t.Fatalf("Status() = %q, want %q", trip.Status(), domaintrip.Completed)
		}

		if trip.FinishedAt() == nil || !trip.FinishedAt().Equal(finishedAt) {
			t.Fatalf("FinishedAt() = %v, want %v", trip.FinishedAt(), finishedAt)
		}
	})

	t.Run("completed trip", func(t *testing.T) {
		trip := domaintrip.RestoreTrip(
			uuid.New(),
			uuid.New(),
			uuid.New(),
			1000,
			domaintrip.Completed,
			domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
			domaintrip.Point{Latitude: 59.93, Longitude: 30.31},
			&startedAt,
			&finishedAt,
			&startedAt,
		)

		err := trip.Finish(finishedAt.Add(time.Hour))
		if !errors.Is(err, domaintrip.ErrTripCompleted) {
			t.Fatalf("Finish() error = %v, want %v", err, domaintrip.ErrTripCompleted)
		}
	})
}

func TestTripSameRequest(t *testing.T) {
	startedAt := time.Now()
	userID := uuid.New()
	driverID := uuid.New()

	first := mustCreateTrip(
		t,
		uuid.New(),
		userID,
		driverID,
		1000,
		domaintrip.Point{Latitude: 1, Longitude: 2},
		domaintrip.Point{Latitude: 3, Longitude: 4},
		startedAt,
	)

	same := mustCreateTrip(
		t,
		uuid.New(),
		userID,
		driverID,
		1000,
		domaintrip.Point{Latitude: 1, Longitude: 2},
		domaintrip.Point{Latitude: 3, Longitude: 4},
		startedAt,
	)

	different := mustCreateTrip(
		t,
		uuid.New(),
		userID,
		driverID,
		2000,
		domaintrip.Point{Latitude: 1, Longitude: 2},
		domaintrip.Point{Latitude: 3, Longitude: 4},
		startedAt,
	)

	if !first.SameRequest(same) {
		t.Fatal("SameRequest() = false for equal requests")
	}

	if first.SameRequest(different) {
		t.Fatal("SameRequest() = true for requests with different price")
	}
}

func TestCreateTripValidation(t *testing.T) {
	validStart := domaintrip.Point{Latitude: 55.75, Longitude: 37.61}
	validEnd := domaintrip.Point{Latitude: 59.93, Longitude: 30.31}

	tests := []struct {
		name       string
		price      int64
		startPoint domaintrip.Point
		endPoint   domaintrip.Point
		wantErr    error
	}{
		{
			name:       "negative price",
			price:      -1,
			startPoint: validStart,
			endPoint:   validEnd,
			wantErr:    domaintrip.ErrInvalidPrice,
		},
		{
			name:       "latitude out of range",
			price:      1000,
			startPoint: domaintrip.Point{Latitude: 91, Longitude: 37.61},
			endPoint:   validEnd,
			wantErr:    domaintrip.ErrInvalidCoordinates,
		},
		{
			name:       "longitude out of range",
			price:      1000,
			startPoint: validStart,
			endPoint:   domaintrip.Point{Latitude: 59.93, Longitude: 181},
			wantErr:    domaintrip.ErrInvalidCoordinates,
		},
		{
			name:       "NaN coordinate",
			price:      1000,
			startPoint: domaintrip.Point{Latitude: math.NaN(), Longitude: 37.61},
			endPoint:   validEnd,
			wantErr:    domaintrip.ErrInvalidCoordinates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip, err := domaintrip.CreateTrip(
				uuid.New(),
				uuid.New(),
				uuid.New(),
				tt.price,
				tt.startPoint,
				tt.endPoint,
				time.Now(),
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateTrip() error = %v, want %v", err, tt.wantErr)
			}
			if trip != nil {
				t.Fatalf("CreateTrip() trip = %#v, want nil", trip)
			}
		})
	}
}
