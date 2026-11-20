package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type Service struct {
	tripRepo tripRepository
}

func NewService(
	tripRepo tripRepository,
) *Service {
	return &Service{
		tripRepo: tripRepo,
	}
}

func (s *Service) GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	trip, err := s.tripRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return trip, nil
}
