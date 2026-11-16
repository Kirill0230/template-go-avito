//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test

package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type usecase interface {
	CreateTrip(ctx context.Context, trip model.Trip, idempotencyKey *uuid.UUID) (*model.Trip, bool, error)
}
