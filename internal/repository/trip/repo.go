package trip

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
	psql    sq.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool, timeout time.Duration) *TripRepository {
	return &TripRepository{
		pool:    pool,
		timeout: timeout,
		psql:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

var tripColumns = []string{
	"id",
	"user_id",
	"driver_id",
	"start_latitude",
	"start_longitude",
	"end_latitude",
	"end_longitude",
	"price",
	"status",
	"started_at",
	"finished_at",
}
