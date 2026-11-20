package create

import (
	"context"
	"time"

	"github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
	tripmapper "github.com/Kirill0230/template-go-avito/internal/usecase/trip"
)

type Service struct {
	trips     tripRepository
	positions positionRepository
	tx        txManager
	pending   pendingCanceler
	metrics   positionMetrics
	now       func() time.Time
}

func NewService(
	trips tripRepository,
	positions positionRepository,
	tx txManager,
	pending pendingCanceler,
	metrics positionMetrics,
	now func() time.Time,
) *Service {
	return &Service{
		trips: trips, positions: positions, tx: tx,
		pending: pending, metrics: metrics, now: now,
	}
}

func (s *Service) Create(ctx context.Context, position model.TripPosition) (*model.TripPosition, error) {
	var saved *model.TripPosition
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		modelTrip, err := s.trips.GetForUpdate(ctx, position.TripID)
		if err != nil {
			return err
		}

		domainTrip, err := tripmapper.ModelToDomain(*modelTrip)
		if err != nil {
			return err
		}
		if err := domainTrip.AcceptPosition(
			trip.Point{
				Latitude:  position.Coordinates.Latitude,
				Longitude: position.Coordinates.Longitude,
			},
			position.RecordedAt,
			s.now(),
		); err != nil {
			return err
		}

		saved, err = s.positions.Save(ctx, position)
		if err != nil {
			return err
		}
		return s.trips.UpdateLastPositionAt(ctx, position.TripID, position.RecordedAt)
	})
	if err != nil {
		return nil, err
	}

	s.pending.Cancel(position.TripID)
	s.metrics.PositionSaved(ctx)
	return saved, nil
}
