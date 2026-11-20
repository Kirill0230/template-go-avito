package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type Service struct {
	trips     tripRepository
	positions positionRepository
}

func NewService(trips tripRepository, positions positionRepository) *Service {
	return &Service{trips: trips, positions: positions}
}

func (s *Service) List(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error) {
	if _, err := s.trips.Get(ctx, tripID); err != nil {
		return nil, err
	}

	return s.positions.List(ctx, tripID)
}
