package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

func (r *TripRepository) Save(ctx context.Context, modelTrip model.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	query, args, err := r.psql.Insert("trips").Columns(tripColumns...).Values(
		modelTrip.Id,
		modelTrip.UserId,
		modelTrip.DriverId,
		modelTrip.StartPoint.Latitude,
		modelTrip.StartPoint.Longitude,
		modelTrip.EndPoint.Latitude,
		modelTrip.EndPoint.Longitude,
		modelTrip.Price,
		string(modelTrip.Status),
		modelTrip.StartedAt,
		modelTrip.FinishedAt,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build add trip query: %w", err)
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
