package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
)

func TestTripFinish(t *testing.T) {
	startedAt := time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)

	t.Run("active trip", func(t *testing.T) {
		trip := domaintrip.CreateTrip(
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

	first := domaintrip.CreateTrip(
		uuid.New(),
		userID,
		driverID,
		1000,
		domaintrip.Point{Latitude: 1, Longitude: 2},
		domaintrip.Point{Latitude: 3, Longitude: 4},
		startedAt,
	)

	same := domaintrip.CreateTrip(
		uuid.New(),
		userID,
		driverID,
		1000,
		domaintrip.Point{Latitude: 1, Longitude: 2},
		domaintrip.Point{Latitude: 3, Longitude: 4},
		startedAt,
	)

	different := domaintrip.CreateTrip(
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
