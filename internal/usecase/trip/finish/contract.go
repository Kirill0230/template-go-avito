//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type (
	tripRepository interface {
		GetForUpdate(ctx context.Context, id uuid.UUID) (*model.Trip, error)
		Update(ctx context.Context, modelTrip model.Trip) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripStatusHistoryRepository interface {
		Add(ctx context.Context, tripID uuid.UUID, from, to model.Status, changedAt time.Time) error
	}

	tripMetrics interface {
		TripCompleted(ctx context.Context)
	}
)
