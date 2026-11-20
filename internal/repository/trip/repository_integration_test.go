//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
	positionrepo "github.com/Kirill0230/template-go-avito/internal/repository/position"
	triprepo "github.com/Kirill0230/template-go-avito/internal/repository/trip"
	statusrepo "github.com/Kirill0230/template-go-avito/internal/repository/trip_status_history"
)

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func repositoryTrip(id uuid.UUID) model.Trip {
	startedAt := time.Now().UTC().Truncate(time.Microsecond)

	return model.Trip{
		Id:        id,
		UserId:    uuid.New(),
		DriverId:  uuid.New(),
		Price:     1000,
		Status:    model.Active,
		StartedAt: &startedAt,
		StartPoint: model.Point{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: model.Point{
			Latitude:  59.93,
			Longitude: 30.31,
		},
	}
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			id,
		)
	})
}

func TestTripRepositorySaveAndGet(t *testing.T) {
	pool := openPool(t)
	repository := triprepo.NewTripRepository(pool, 3*time.Second)

	tripID := uuid.New()
	cleanupTrip(t, pool, tripID)

	expected := repositoryTrip(tripID)

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	actual, err := repository.Get(context.Background(), tripID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if actual.Id != expected.Id {
		t.Errorf("trip ID = %s, want %s", actual.Id, expected.Id)
	}

	if actual.Status != model.Active {
		t.Errorf("status = %q, want %q", actual.Status, model.Active)
	}
}

func TestTransactionRollback(t *testing.T) {
	pool := openPool(t)

	tripRepository := triprepo.NewTripRepository(pool, 3*time.Second)
	historyRepository := statusrepo.NewTripStatusHistoryRepository(pool, 3*time.Second)
	txManager := transaction.NewTxManagerImpl(pool, 3*time.Second)

	tripID := uuid.New()
	cleanupTrip(t, pool, tripID)

	err := txManager.Do(context.Background(), func(ctx context.Context) error {
		if err := tripRepository.Save(ctx, repositoryTrip(tripID)); err != nil {
			return err
		}

		return historyRepository.Add(
			ctx,
			tripID,
			"",
			model.Status("invalid"),
			time.Now(),
		)
	})
	if err == nil {
		t.Fatal("transaction error is nil")
	}

	_, err = tripRepository.Get(context.Background(), tripID)
	if !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Fatalf("trip was not rolled back: %v", err)
	}
}

func TestListStaleTrips(t *testing.T) {
	pool := openPool(t)
	repository := triprepo.NewTripRepository(pool, 3*time.Second)

	stale := repositoryTrip(uuid.New())
	fresh := repositoryTrip(uuid.New())
	completed := repositoryTrip(uuid.New())
	old := time.Now().UTC().Add(-10 * time.Minute)
	now := time.Now().UTC()
	stale.StartedAt = &old
	fresh.StartedAt = &now
	completed.StartedAt = &old
	completed.Status = model.Completed
	completed.FinishedAt = &now

	for _, trip := range []model.Trip{stale, fresh, completed} {
		cleanupTrip(t, pool, trip.Id)
		if err := repository.Save(context.Background(), trip); err != nil {
			t.Fatalf("Save(%s) error: %v", trip.Id, err)
		}
	}

	got, err := repository.ListStale(context.Background(), time.Minute, nil, 100)
	if err != nil {
		t.Fatalf("ListStale() error: %v", err)
	}
	found := make(map[uuid.UUID]bool)
	for _, trip := range got {
		found[trip.ID] = true
	}
	if !found[stale.Id] || found[fresh.Id] || found[completed.Id] {
		t.Fatalf("unexpected stale set: stale=%v fresh=%v completed=%v", found[stale.Id], found[fresh.Id], found[completed.Id])
	}
}

func TestPositionAndLastPositionAtAreAtomic(t *testing.T) {
	pool := openPool(t)
	trips := triprepo.NewTripRepository(pool, 3*time.Second)
	positions := positionrepo.NewRepository(pool, 3*time.Second)
	txManager := transaction.NewTxManagerImpl(pool, 3*time.Second)
	trip := repositoryTrip(uuid.New())
	cleanupTrip(t, pool, trip.Id)
	if err := trips.Save(context.Background(), trip); err != nil {
		t.Fatal(err)
	}
	recordedAt := time.Now().UTC().Truncate(time.Microsecond)

	err := txManager.Do(context.Background(), func(ctx context.Context) error {
		if _, err := positions.Save(ctx, model.TripPosition{
			TripID: trip.Id, Coordinates: model.Point{Latitude: 59.93, Longitude: 30.33}, RecordedAt: recordedAt,
		}); err != nil {
			return err
		}
		if err := trips.UpdateLastPositionAt(ctx, trip.Id, recordedAt); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("transaction unexpectedly succeeded")
	}

	gotPositions, err := positions.List(context.Background(), trip.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotPositions) != 0 {
		t.Fatalf("positions after rollback = %d", len(gotPositions))
	}
	gotTrip, err := trips.Get(context.Background(), trip.Id)
	if err != nil {
		t.Fatal(err)
	}
	if gotTrip.LastPositionAt != nil {
		t.Fatalf("last_position_at after rollback = %v", gotTrip.LastPositionAt)
	}

	for _, at := range []time.Time{recordedAt, recordedAt.Add(-time.Minute)} {
		err = txManager.Do(context.Background(), func(ctx context.Context) error {
			if _, err := positions.Save(ctx, model.TripPosition{
				TripID: trip.Id, Coordinates: model.Point{Latitude: 59.93, Longitude: 30.33}, RecordedAt: at,
			}); err != nil {
				return err
			}
			return trips.UpdateLastPositionAt(ctx, trip.Id, at)
		})
		if err != nil {
			t.Fatalf("save position at %v: %v", at, err)
		}
	}
	gotTrip, err = trips.Get(context.Background(), trip.Id)
	if err != nil {
		t.Fatal(err)
	}
	if gotTrip.LastPositionAt == nil || !gotTrip.LastPositionAt.Equal(recordedAt) {
		t.Fatalf("last_position_at = %v, want %v", gotTrip.LastPositionAt, recordedAt)
	}
}
