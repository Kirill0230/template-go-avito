//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type usecase interface {
	FinishTrip(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}
