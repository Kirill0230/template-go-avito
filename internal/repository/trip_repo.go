package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kirill0230/template-go-avito/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
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

type tripRow struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	DriverID       uuid.UUID  `db:"driver_id"`
	StartLatitude  float64    `db:"start_latitude"`
	StartLongitude float64    `db:"start_longitude"`
	EndLatitude    float64    `db:"end_latitude"`
	EndLongitude   float64    `db:"end_longitude"`
	Price          int64      `db:"price"`
	Status         string     `db:"status"`
	StartedAt      time.Time  `db:"started_at"`
	FinishedAt     *time.Time `db:"finished_at"`
}

func (r *TripRepository) GetTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	query, args, err := psql.Select(tripColumns...).From("trips").Where(sq.Eq{"id": id}).ToSql()

	if err != nil {
		return nil, fmt.Errorf("build get trip query: %w", err)
	}

	rows, err := executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[tripRow])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTripNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return row.toDomain(), nil
}

func (r *TripRepository) AddTrip(ctx context.Context, trip domain.Trip) error {
	query, args, err := psql.Insert("trips").Columns(tripColumns...).Values(
		trip.Id,
		trip.UserId,
		trip.DriverId,
		trip.StartPoint.Latitude,
		trip.StartPoint.Longitude,
		trip.EndPoint.Latitude,
		trip.EndPoint.Longitude,
		trip.Price,
		string(trip.Status),
		trip.StartedAt,
		trip.FinishedAt,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build add trip query: %w", err)
	}

	_, err = executor(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_status_active_at_idx" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("add trip: %w", err)
	}

	return nil
}

func (r *TripRepository) FinishTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*domain.Trip, error) {
	query, args, err := psql.
		Update("trips").
		Set("status", string(domain.Completed)).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{
			"id":     id,
			"status": string(domain.Active),
		}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build finish trip query: %w", err)
	}

	rows, err := executor(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[tripRow])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, r.finishFailedReason(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	return row.toDomain(), nil
}

func (r tripRow) toDomain() *domain.Trip {
	startedAt := r.StartedAt

	return &domain.Trip{
		Id:       r.ID,
		UserId:   r.UserID,
		DriverId: r.DriverID,
		StartPoint: domain.Point{
			Latitude:  r.StartLatitude,
			Longitude: r.StartLongitude,
		},
		EndPoint: domain.Point{
			Latitude:  r.EndLatitude,
			Longitude: r.EndLongitude,
		},
		Price:      r.Price,
		Status:     domain.Status(r.Status),
		StartedAt:  &startedAt,
		FinishedAt: r.FinishedAt,
	}
}

func (r *TripRepository) finishFailedReason(ctx context.Context, id uuid.UUID) error {
	_, err := r.GetTrip(ctx, id)
	if errors.Is(err, domain.ErrTripNotFound) {
		return domain.ErrTripNotFound
	}
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}
	return domain.ErrTripAlreadyComplete
}
