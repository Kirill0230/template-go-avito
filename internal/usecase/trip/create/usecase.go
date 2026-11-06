package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
	tripmapper "github.com/Kirill0230/template-go-avito/internal/usecase/trip"
)

type Service struct {
	tripRepo              tripRepository
	tripStatusHistoryRepo tripStatusHistoryRepository
	idempotencyRepo       idempotencyKeyRepository
	tx                    txManager
	metrics               tripMetrics
}

func NewService(
	tripRepo tripRepository,
	tripStatusHistoryRepo tripStatusHistoryRepository,
	idempotencyRepo idempotencyKeyRepository,
	tx txManager,
	metrics tripMetrics,
) *Service {
	return &Service{
		tripRepo:              tripRepo,
		tripStatusHistoryRepo: tripStatusHistoryRepo,
		idempotencyRepo:       idempotencyRepo,
		tx:                    tx,
		metrics:               metrics,
	}
}

func (s *Service) CreateTrip(ctx context.Context, modelTrip model.Trip, idempotencyKey *uuid.UUID) (*model.Trip, bool, error) {
	resultTrip := &modelTrip
	created := true

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		newTripId := uuid.New()

		now := time.Now()

		modelTrip.Id = newTripId
		modelTrip.Status = model.Active
		modelTrip.StartedAt = &now
		modelTrip.LastPositionAt = &now

		if idempotencyKey != nil {
			claimed, err := s.idempotencyRepo.Claim(ctx, *idempotencyKey, newTripId)
			if err != nil {
				return err
			}
			if !claimed {
				tripID, err := s.idempotencyRepo.Get(ctx, *idempotencyKey)
				if err != nil {
					return err
				}
				existing, err := s.tripRepo.Get(ctx, tripID)
				if err != nil {
					return err
				}

				alreadyTrip, err := tripmapper.ModelToDomain(*existing)
				if err != nil {
					return err
				}

				domainTrip, err := tripmapper.ModelToDomain(modelTrip)
				if err != nil {
					return err
				}

				if !alreadyTrip.SameRequest(domainTrip) {
					return domaintrip.ErrIdempotencyConflict
				}

				mappedTrip := tripmapper.DomainToModel(alreadyTrip)
				resultTrip = &mappedTrip

				created = false
				return nil
			}
		}

		newTrip := domaintrip.CreateTrip(
			modelTrip.Id,
			modelTrip.UserId,
			modelTrip.DriverId,
			modelTrip.Price,
			domaintrip.Point{
				Latitude:  modelTrip.StartPoint.Latitude,
				Longitude: modelTrip.StartPoint.Longitude,
			},
			domaintrip.Point{
				Latitude:  modelTrip.EndPoint.Latitude,
				Longitude: modelTrip.EndPoint.Longitude,
			},
			*modelTrip.StartedAt,
		)

		mappedTrip := tripmapper.DomainToModel(newTrip)

		err := s.tripRepo.Save(ctx, mappedTrip)
		if err != nil {
			return err
		}

		resultTrip = &mappedTrip

		return s.tripStatusHistoryRepo.Add(ctx, newTrip.Id(), "", model.Active, now)
	})
	if err != nil {
		return nil, false, err
	}

	if created {
		s.metrics.TripCreated(ctx)
	}

	return resultTrip, created, nil
}
