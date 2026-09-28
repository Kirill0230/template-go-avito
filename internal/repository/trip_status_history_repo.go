package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Kirill0230/template-go-avito/internal/domain"
)

type TripStatusHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewTripStatusHistoryRepository(pool *pgxpool.Pool) *TripStatusHistoryRepository {
	return &TripStatusHistoryRepository{pool: pool}
}

func (r *TripStatusHistoryRepository) Add(ctx context.Context, tripID uuid.UUID, from domain.Status, to domain.Status, changedAt time.Time) error {
	var fromStatus *string
	if from != "" {
		s := string(from)
		fromStatus = &s
	}

	query, args, err := psql.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"changed_at",
		).
		Values(
			tripID,
			fromStatus,
			string(to),
			changedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add status history query: %w", err)
	}

	_, err = executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add status history: %w", err)
	}

	return nil
}
