package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

const firstStaleTripsBatchQuery = `
SELECT id, driver_id, COALESCE(last_position_at, started_at) AS silent_since
FROM trips
WHERE status = 'active'
  AND COALESCE(last_position_at, started_at) < now() - $1::bigint * interval '1 microsecond'
ORDER BY COALESCE(last_position_at, started_at), id
LIMIT $2`

const nextStaleTripsBatchQuery = `
SELECT id, driver_id, COALESCE(last_position_at, started_at) AS silent_since
FROM trips
WHERE status = 'active'
  AND COALESCE(last_position_at, started_at) < now() - $1::bigint * interval '1 microsecond'
  AND (COALESCE(last_position_at, started_at), id) > ($2, $3)
ORDER BY COALESCE(last_position_at, started_at), id
LIMIT $4`

type staleTripRow struct {
	ID          uuid.UUID `db:"id"`
	DriverID    uuid.UUID `db:"driver_id"`
	SilentSince time.Time `db:"silent_since"`
}

func (r *TripRepository) ListStale(
	ctx context.Context,
	staleAfter time.Duration,
	cursor *model.StaleTripCursor,
	limit int,
) ([]model.StaleTrip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	ctx = postgres.WithDBCollection(ctx, "trips")

	query := firstStaleTripsBatchQuery
	args := []any{staleAfter.Microseconds(), limit}
	if cursor != nil {
		query = nextStaleTripsBatchQuery
		args = []any{staleAfter.Microseconds(), cursor.SilentSince, cursor.ID, limit}
	}

	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stale trips batch: %w", err)
	}

	rowTrips, err := pgx.CollectRows(rows, pgx.RowToStructByName[staleTripRow])
	if err != nil {
		return nil, fmt.Errorf("collect stale trips batch: %w", err)
	}

	trips := make([]model.StaleTrip, 0, len(rowTrips))
	for _, trip := range rowTrips {
		trips = append(trips, model.StaleTrip{
			ID: trip.ID, DriverID: trip.DriverID, SilentSince: trip.SilentSince,
		})
	}

	return trips, nil
}
