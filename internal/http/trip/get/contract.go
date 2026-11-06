//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test

package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type usecase interface {
	GetTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}
