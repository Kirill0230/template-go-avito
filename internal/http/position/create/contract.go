//go:generate go tool mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type usecase interface {
	Create(ctx context.Context, position model.TripPosition) (*model.TripPosition, error)
}
