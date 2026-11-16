package idempotency

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyKeyRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
	psql    sq.StatementBuilderType
}

func NewIdempotencyKeyRepository(pool *pgxpool.Pool, timeout time.Duration) *IdempotencyKeyRepository {
	return &IdempotencyKeyRepository{
		pool:    pool,
		timeout: timeout,
		psql:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

const idempotencyKeyTTL = 24 * time.Hour
