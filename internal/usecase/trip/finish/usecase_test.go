package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
	finishusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/finish"
)

func activeTrip(id uuid.UUID) *model.Trip {
	startedAt := time.Now()

	return &model.Trip{
		Id:             id,
		UserId:         uuid.New(),
		DriverId:       uuid.New(),
		Price:          1000,
		Status:         model.Active,
		StartedAt:      &startedAt,
		LastPositionAt: &startedAt,
		StartPoint:     model.Point{Latitude: 1, Longitude: 2},
		EndPoint:       model.Point{Latitude: 3, Longitude: 4},
	}
}

func expectFinishTransaction(tx *MocktxManager) {
	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
}

func TestFinishTripSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	tripRepo := NewMocktripRepository(ctrl)
	historyRepo := NewMocktripStatusHistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	metrics := NewMocktripMetrics(ctrl)

	tripID := uuid.New()

	expectFinishTransaction(tx)

	tripRepo.EXPECT().
		GetForUpdate(gomock.Any(), tripID).
		Return(activeTrip(tripID), nil)

	tripRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, trip model.Trip) error {
			if trip.Status != model.Completed {
				t.Errorf("updated status = %q, want %q", trip.Status, model.Completed)
			}

			if trip.FinishedAt == nil {
				t.Error("updated FinishedAt is nil")
			}

			return nil
		})

	historyRepo.EXPECT().
		Add(
			gomock.Any(),
			tripID,
			model.Active,
			model.Completed,
			gomock.Any(),
		).
		Return(nil)

	metrics.EXPECT().
		TripCompleted(gomock.Any())

	service := finishusecase.NewService(
		tripRepo,
		historyRepo,
		tx,
		metrics,
	)

	result, err := service.FinishTrip(context.Background(), tripID)
	if err != nil {
		t.Fatalf("FinishTrip() error: %v", err)
	}

	if result == nil || result.Status != model.Completed {
		t.Fatalf("FinishTrip() result = %#v", result)
	}
}

func TestFinishTripCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)

	tripRepo := NewMocktripRepository(ctrl)
	historyRepo := NewMocktripStatusHistoryRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	metrics := NewMocktripMetrics(ctrl)

	tripID := uuid.New()
	trip := activeTrip(tripID)
	finishedAt := time.Now()

	trip.Status = model.Completed
	trip.FinishedAt = &finishedAt

	expectFinishTransaction(tx)

	tripRepo.EXPECT().
		GetForUpdate(gomock.Any(), tripID).
		Return(trip, nil)

	service := finishusecase.NewService(
		tripRepo,
		historyRepo,
		tx,
		metrics,
	)

	result, err := service.FinishTrip(context.Background(), tripID)
	if !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Fatalf("FinishTrip() error = %v, want %v", err, domaintrip.ErrTripCompleted)
	}

	if result != nil {
		t.Fatalf("FinishTrip() result = %#v, want nil", result)
	}
}
