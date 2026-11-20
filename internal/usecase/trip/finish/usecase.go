package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
	tripmapper "github.com/Kirill0230/template-go-avito/internal/usecase/trip"
)

type Service struct {
	tripRepo              tripRepository
	tripStatusHistoryRepo tripStatusHistoryRepository
	tx                    txManager
	metrics               tripMetrics
}

func NewService(
	tripRepo tripRepository,
	tripStatusHistoryRepo tripStatusHistoryRepository,
	tx txManager,
	metrics tripMetrics,
) *Service {
	return &Service{
		tripRepo:              tripRepo,
		tripStatusHistoryRepo: tripStatusHistoryRepo,
		tx:                    tx,
		metrics:               metrics,
	}
}

func (s *Service) FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	var resultTrip *model.Trip

	nowTime := time.Now()

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		finishTrip, err := s.tripRepo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}

		domainTrip, err := tripmapper.ModelToDomain(*finishTrip)
		if err != nil {
			return err
		}

		err = domainTrip.Finish(nowTime)
		if err != nil {
			return err
		}

		err = s.tripRepo.Update(ctx, tripmapper.DomainToModel(domainTrip))
		if err != nil {
			return err
		}

		err = s.tripStatusHistoryRepo.Add(ctx, id, model.Active, model.Completed, nowTime)
		if err != nil {
			return err
		}

		mappedTrip := tripmapper.DomainToModel(domainTrip)
		resultTrip = &mappedTrip
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.metrics.TripCompleted(ctx)

	return resultTrip, nil
}
