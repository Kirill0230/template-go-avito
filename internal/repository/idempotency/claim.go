package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
)

func (r *IdempotencyKeyRepository) Claim(ctx context.Context, key, tripID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	expiredBefore := time.Now().Add(-idempotencyKeyTTL)
	query, args, err := r.psql.
		Insert("idempotency_keys").
		Columns("key", "trip_id").
		Values(key, tripID).
		Suffix(`ON CONFLICT (key) DO UPDATE
			SET trip_id    = EXCLUDED.trip_id,
			    created_at = now()
			WHERE idempotency_keys.created_at < ?
			RETURNING key`, expiredBefore).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build claim idempotency key query: %w", err)
	}

	var claimed uuid.UUID
	err = transaction.Executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim idempotency key: %w", err)
	}
	return true, nil
}
