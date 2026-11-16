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
