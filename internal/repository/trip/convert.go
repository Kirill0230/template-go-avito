package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
)

type tripRow struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	DriverID       uuid.UUID  `db:"driver_id"`
	StartLatitude  float64    `db:"start_latitude"`
	StartLongitude float64    `db:"start_longitude"`
	EndLatitude    float64    `db:"end_latitude"`
	EndLongitude   float64    `db:"end_longitude"`
	Price          int64      `db:"price"`
	Status         string     `db:"status"`
	StartedAt      time.Time  `db:"started_at"`
	FinishedAt     *time.Time `db:"finished_at"`
}

func (r tripRow) toModel() *model.Trip {
	startedAt := r.StartedAt

	return &model.Trip{
		Id:       r.ID,
		UserId:   r.UserID,
		DriverId: r.DriverID,
		StartPoint: model.Point{
			Latitude:  r.StartLatitude,
			Longitude: r.StartLongitude,
		},
		EndPoint: model.Point{
			Latitude:  r.EndLatitude,
			Longitude: r.EndLongitude,
		},
		Price:      r.Price,
		Status:     model.Status(r.Status),
		StartedAt:  &startedAt,
		FinishedAt: r.FinishedAt,
	}
}
