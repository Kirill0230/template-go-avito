//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type (
	tripRepository interface {
		Save(ctx context.Context, trip model.Trip) error
		Get(ctx context.Context, id uuid.UUID) (*model.Trip, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripStatusHistoryRepository interface {
		Add(ctx context.Context, tripID uuid.UUID, from, to model.Status, changedAt time.Time) error
	}

	idempotencyKeyRepository interface {
		Get(ctx context.Context, key uuid.UUID) (uuid.UUID, error)
		Claim(ctx context.Context, key, tripID uuid.UUID) (bool, error)
	}

	tripMetrics interface {
		TripCreated(ctx context.Context)
	}
)
