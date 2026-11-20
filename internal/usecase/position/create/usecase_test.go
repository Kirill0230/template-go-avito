package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Kirill0230/template-go-avito/internal/model"
	positioncreate "github.com/Kirill0230/template-go-avito/internal/usecase/position/create"
)

func runTransaction(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
}

func activeTrip(id uuid.UUID, startedAt time.Time) *model.Trip {
	return &model.Trip{
		Id: id, UserId: uuid.New(), DriverId: uuid.New(), Price: 1000,
		Status: model.Active, StartedAt: &startedAt,
		StartPoint: model.Point{Latitude: 55, Longitude: 37},
		EndPoint:   model.Point{Latitude: 56, Longitude: 38},
	}
}

func TestCreatePositionCommitsAndCancelsPendingTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	positions := NewMockpositionRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	pending := NewMockpendingCanceler(ctrl)
	metrics := NewMockpositionMetrics(ctrl)
	now := time.Date(2026, time.November, 20, 12, 0, 0, 0, time.UTC)
	position := model.TripPosition{
		TripID: uuid.New(), Coordinates: model.Point{Latitude: 59.93, Longitude: 30.33}, RecordedAt: now,
	}
	saved := position
	saved.ID = 42

	runTransaction(tx)
	trips.EXPECT().GetForUpdate(gomock.Any(), position.TripID).Return(activeTrip(position.TripID, now.Add(-time.Hour)), nil)
	positions.EXPECT().Save(gomock.Any(), position).Return(&saved, nil)
	trips.EXPECT().UpdateLastPositionAt(gomock.Any(), position.TripID, position.RecordedAt).Return(nil)
	pending.EXPECT().Cancel(position.TripID)
	metrics.EXPECT().PositionSaved(gomock.Any())

	service := positioncreate.NewService(trips, positions, tx, pending, metrics, func() time.Time { return now })
	got, err := service.Create(context.Background(), position)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if got.ID != saved.ID {
		t.Fatalf("Create() ID = %d, want %d", got.ID, saved.ID)
	}
}

func TestCreatePositionDoesNotCancelTaskOnRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	trips := NewMocktripRepository(ctrl)
	positions := NewMockpositionRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	pending := NewMockpendingCanceler(ctrl)
	metrics := NewMockpositionMetrics(ctrl)
	now := time.Now()
	position := model.TripPosition{TripID: uuid.New(), Coordinates: model.Point{}, RecordedAt: now}
	wantErr := errors.New("save failed")

	runTransaction(tx)
	trips.EXPECT().GetForUpdate(gomock.Any(), position.TripID).Return(activeTrip(position.TripID, now), nil)
	positions.EXPECT().Save(gomock.Any(), position).Return(nil, wantErr)

	service := positioncreate.NewService(trips, positions, tx, pending, metrics, func() time.Time { return now })
	if _, err := service.Create(context.Background(), position); !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
}
