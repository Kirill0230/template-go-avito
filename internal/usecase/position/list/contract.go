//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}

type positionRepository interface {
	List(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error)
}
