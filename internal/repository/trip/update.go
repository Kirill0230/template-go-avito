package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

func (r *TripRepository) Update(ctx context.Context, modelTrip model.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := r.psql.Update("trips").
		Set("status", string(modelTrip.Status)).
		Set("price", modelTrip.Price).
		Set("finished_at", modelTrip.FinishedAt).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": modelTrip.Id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip query: %w", err)
	}

	_, err = transaction.Executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_status_active_at_idx" {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("add trip: %w", err)
	}

	return nil
}
