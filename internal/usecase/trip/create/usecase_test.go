package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
	createusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/create"
)

func testCreateTrip() model.Trip {
	return model.Trip{
		UserId:   uuid.New(),
		DriverId: uuid.New(),
		Price:    1000,
		StartPoint: model.Point{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: model.Point{
			Latitude:  59.93,
			Longitude: 30.31,
		},
	}
}

func expectTransaction(tx *MocktxManager) {
	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
}

func TestCreateTripSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	tripRepo := NewMocktripRepository(ctrl)
	historyRepo := NewMocktripStatusHistoryRepository(ctrl)
	idempotencyRepo := NewMockidempotencyKeyRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	metrics := NewMocktripMetrics(ctrl)

	expectTransaction(tx)

	tripRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, trip model.Trip) error {
			if trip.Id == uuid.Nil {
				t.Error("saved trip ID is empty")
			}

			if trip.Status != model.Active {
				t.Errorf("saved status = %q, want %q", trip.Status, model.Active)
			}

			return nil
		})

	historyRepo.EXPECT().
		Add(
			gomock.Any(),
			gomock.Any(),
			model.Status(""),
			model.Active,
			gomock.Any(),
		).
		Return(nil)

	metrics.EXPECT().
		TripCreated(gomock.Any())

	service := createusecase.NewService(
		tripRepo,
		historyRepo,
		idempotencyRepo,
		tx,
		metrics,
	)

	result, created, err := service.CreateTrip(
		context.Background(),
		testCreateTrip(),
		nil,
	)
	if err != nil {
		t.Fatalf("CreateTrip() error: %v", err)
	}

	if !created {
		t.Fatal("CreateTrip() created = false, want true")
	}

	if result == nil || result.Status != model.Active {
		t.Fatalf("CreateTrip() result = %#v", result)
	}
}

func TestCreateTripDriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)

	tripRepo := NewMocktripRepository(ctrl)
	historyRepo := NewMocktripStatusHistoryRepository(ctrl)
	idempotencyRepo := NewMockidempotencyKeyRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	metrics := NewMocktripMetrics(ctrl)

	expectTransaction(tx)

	tripRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(domaintrip.ErrDriverBusy)

	service := createusecase.NewService(
		tripRepo,
		historyRepo,
		idempotencyRepo,
		tx,
		metrics,
	)

	result, created, err := service.CreateTrip(
		context.Background(),
		testCreateTrip(),
		nil,
	)

	if !errors.Is(err, domaintrip.ErrDriverBusy) {
		t.Fatalf("CreateTrip() error = %v, want %v", err, domaintrip.ErrDriverBusy)
	}

	if result != nil {
		t.Fatalf("CreateTrip() result = %#v, want nil", result)
	}

	if created {
		t.Fatal("CreateTrip() created = true, want false")
	}
}

func TestCreateTripIdempotentReplay(t *testing.T) {
	ctrl := gomock.NewController(t)

	tripRepo := NewMocktripRepository(ctrl)
	historyRepo := NewMocktripStatusHistoryRepository(ctrl)
	idempotencyRepo := NewMockidempotencyKeyRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	metrics := NewMocktripMetrics(ctrl)

	idempotencyKey := uuid.New()
	existingTripID := uuid.New()
	startedAt := time.Now()

	requestedTrip := testCreateTrip()
	existingTrip := requestedTrip
	existingTrip.Id = existingTripID
	existingTrip.Status = model.Active
	existingTrip.StartedAt = &startedAt
	existingTrip.LastPositionAt = &startedAt

	expectTransaction(tx)

	idempotencyRepo.EXPECT().
		Claim(gomock.Any(), idempotencyKey, gomock.Any()).
		Return(false, nil)

	idempotencyRepo.EXPECT().
		Get(gomock.Any(), idempotencyKey).
		Return(existingTripID, nil)

	tripRepo.EXPECT().
		Get(gomock.Any(), existingTripID).
		Return(&existingTrip, nil)

	service := createusecase.NewService(
		tripRepo,
		historyRepo,
		idempotencyRepo,
		tx,
		metrics,
	)

	result, created, err := service.CreateTrip(
		context.Background(),
		requestedTrip,
		&idempotencyKey,
	)
	if err != nil {
		t.Fatalf("CreateTrip() error: %v", err)
	}

	if created {
		t.Fatal("CreateTrip() created = true, want false")
	}

	if result == nil {
		t.Fatal("CreateTrip() result is nil")
	}

	if result.Id != existingTripID {
		t.Fatalf("CreateTrip() trip ID = %s, want %s", result.Id, existingTripID)
	}
}
