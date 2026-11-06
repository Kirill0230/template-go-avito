package trip_status_history

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

func (r *TripStatusHistoryRepository) Add(ctx context.Context, tripID uuid.UUID, from, to model.Status, changedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var fromStatus *string
	if from != "" {
		s := string(from)
		fromStatus = &s
	}

	query, args, err := r.psql.
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

	_, err = transaction.Executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add status history: %w", err)
	}

	return nil
}
