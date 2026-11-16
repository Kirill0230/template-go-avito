package trip_status_history

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripStatusHistoryRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
	psql    sq.StatementBuilderType
}

func NewTripStatusHistoryRepository(pool *pgxpool.Pool, timeout time.Duration) *TripStatusHistoryRepository {
	return &TripStatusHistoryRepository{
		pool:    pool,
		timeout: timeout,
		psql:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}
