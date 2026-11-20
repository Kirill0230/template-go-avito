package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	getusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/get"
)

func TestGetTripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)

	tripID := uuid.New()

	repo.EXPECT().
		Get(gomock.Any(), tripID).
		Return(nil, domaintrip.ErrTripNotFound)

	service := getusecase.NewService(repo)

	result, err := service.GetTrip(context.Background(), tripID)
	if !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Fatalf("GetTrip() error = %v, want %v", err, domaintrip.ErrTripNotFound)
	}

	if result != nil {
		t.Fatalf("GetTrip() result = %#v, want nil", result)
	}
}
