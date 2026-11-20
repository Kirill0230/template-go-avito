//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type tripRepository interface {
	GetForUpdate(ctx context.Context, id uuid.UUID) (*model.Trip, error)
	UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, recordedAt time.Time) error
}

type positionRepository interface {
	Save(ctx context.Context, position model.TripPosition) (*model.TripPosition, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type pendingCanceler interface {
	Cancel(tripID uuid.UUID)
}

type positionMetrics interface {
	PositionSaved(ctx context.Context)
}
