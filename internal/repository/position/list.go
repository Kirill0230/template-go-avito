package position

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

const listPositionsQuery = `
SELECT id, trip_id, latitude, longitude, recorded_at
FROM trip_positions
WHERE trip_id = $1
ORDER BY recorded_at, id`

type positionRow struct {
	ID         int64     `db:"id"`
	TripID     uuid.UUID `db:"trip_id"`
	Latitude   float64   `db:"latitude"`
	Longitude  float64   `db:"longitude"`
	RecordedAt time.Time `db:"recorded_at"`
}

func (r *Repository) List(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	ctx = postgres.WithDBCollection(ctx, "trip_positions")

	rows, err := transaction.Executor(ctx, r.pool).Query(ctx, listPositionsQuery, tripID)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}

	positionRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[positionRow])
	if err != nil {
		return nil, fmt.Errorf("collect trip positions: %w", err)
	}

	positions := make([]model.TripPosition, 0, len(positionRows))
	for _, row := range positionRows {
		positions = append(positions, model.TripPosition{
			ID:     row.ID,
			TripID: row.TripID,
			Coordinates: model.Point{
				Latitude:  row.Latitude,
				Longitude: row.Longitude,
			},
			RecordedAt: row.RecordedAt,
		})
	}

	return positions, nil
}
