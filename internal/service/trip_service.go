package service

import (
	"context"
	"time"

	"github.com/Kirill0230/template-go-avito/internal/domain"
	"github.com/Kirill0230/template-go-avito/internal/repository"
	"github.com/google/uuid"
)

type TripService struct {
	tripRepo              *repository.TripRepository
	tripStatusHistoryRepo *repository.TripStatusHistoryRepository
	idempotencyRepo       *repository.IdempotencyKeyRepository
	tx                    repository.TxManager
}

func NewTripService(tripRepo *repository.TripRepository,
	tripStatusHistoryRepo *repository.TripStatusHistoryRepository,
	idempotencyRepo *repository.IdempotencyKeyRepository,
	tx repository.TxManager) *TripService {
	return &TripService{
		tripRepo,
		tripStatusHistoryRepo,
		idempotencyRepo,
		tx,
	}
}

func (t *TripService) CreateTrip(ctx context.Context, trip domain.Trip, idempotencyKey *uuid.UUID) (*domain.Trip, bool, error) {
	created := true
	err := t.tx.Do(ctx, func(ctx context.Context) error {
		trip.Id = uuid.New()

		if idempotencyKey != nil {
			claimed, err := t.idempotencyRepo.Claim(ctx, *idempotencyKey, trip.Id)
			if err != nil {
				return err
			}
			if !claimed {
				tripID, err := t.idempotencyRepo.GetTripID(ctx, *idempotencyKey)
				if err != nil {
					return err
				}
				existing, err := t.tripRepo.GetTrip(ctx, tripID)
				if err != nil {
					return err
				}
				if !existing.SameRequest(&trip) {
					return domain.ErrIdempotencyConflict
				}
				trip = *existing
				created = false
				return nil
			}
		}

		now := time.Now()
		trip.Status = domain.Active
		trip.StartedAt = &now

		err := t.tripRepo.AddTrip(ctx, trip)
		if err != nil {
			return err
		}
		return t.tripStatusHistoryRepo.Add(ctx, trip.Id, "", domain.Active, now)
	})
	if err != nil {
		return nil, false, err
	}
	return &trip, created, nil
}

func (t *TripService) GetTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	trip, err := t.tripRepo.GetTrip(ctx, id)
	if err != nil {
		return nil, err
	}

	return trip, nil
}

func (t *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	var trip *domain.Trip
	err := t.tx.Do(ctx, func(ctx context.Context) error {
		now := time.Now()
		finishTrip, err := t.tripRepo.FinishTrip(ctx, id, now)
		if err != nil {
			return err
		}

		err = t.tripStatusHistoryRepo.Add(ctx, id, domain.Active, domain.Completed, now)
		if err != nil {
			return err
		}
		trip = finishTrip
		return nil
	})

	if err != nil {
		return nil, err
	}

	return trip, nil
}
