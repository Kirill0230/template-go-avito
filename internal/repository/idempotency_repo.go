package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyKeyRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewIdempotencyKeyRepository(pool *pgxpool.Pool, timeout time.Duration) *IdempotencyKeyRepository {
	return &IdempotencyKeyRepository{pool: pool, timeout: timeout}
}

const idempotencyKeyTTL = 24 * time.Hour

func (r *IdempotencyKeyRepository) Claim(ctx context.Context, key uuid.UUID, tripID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	expiredBefore := time.Now().Add(-idempotencyKeyTTL)
	query, args, err := psql.
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
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim idempotency key: %w", err)
	}
	return true, nil
}

func (r *IdempotencyKeyRepository) GetTripID(ctx context.Context, key uuid.UUID) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := psql.
		Select("trip_id").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build get idempotency key query: %w", err)
	}

	var tripID uuid.UUID
	err = executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&tripID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return tripID, nil
}
