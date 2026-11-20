package position

import (
	"context"
	"fmt"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

const savePositionQuery = `
INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
VALUES ($1, $2, $3, $4)
RETURNING id`

func (r *Repository) Save(ctx context.Context, position model.TripPosition) (*model.TripPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	ctx = postgres.WithDBCollection(ctx, "trip_positions")

	saved := position
	err := transaction.Executor(ctx, r.pool).QueryRow(
		ctx,
		savePositionQuery,
		position.TripID,
		position.Coordinates.Latitude,
		position.Coordinates.Longitude,
		position.RecordedAt,
	).Scan(&saved.ID)
	if err != nil {
		return nil, fmt.Errorf("save trip position: %w", err)
	}

	return &saved, nil
}
