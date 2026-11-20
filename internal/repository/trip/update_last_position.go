package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
)

const updateLastPositionQuery = `
UPDATE trips
SET last_position_at = GREATEST(
        COALESCE(last_position_at, '-infinity'::timestamptz),
        $2
    ),
    updated_at = now()
WHERE id = $1`

func (r *TripRepository) UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, recordedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	ctx = postgres.WithDBCollection(ctx, "trips")

	if _, err := transaction.Executor(ctx, r.pool).Exec(ctx, updateLastPositionQuery, tripID, recordedAt); err != nil {
		return fmt.Errorf("update trip last position time: %w", err)
	}

	return nil
}
