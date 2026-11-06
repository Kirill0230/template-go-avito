package idempotency

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
)

func (r *IdempotencyKeyRepository) Get(ctx context.Context, key uuid.UUID) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := r.psql.
		Select("trip_id").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build get idempotency key query: %w", err)
	}

	var tripID uuid.UUID
	err = transaction.Executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&tripID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return tripID, nil
}
