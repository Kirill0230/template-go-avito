package trip

import (
	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

func ToAPITrip(t *model.Trip) api.Trip {
	return api.Trip{
		DriverId:       t.DriverId,
		FinishedAt:     t.FinishedAt,
		Id:             t.Id,
		LastPositionAt: t.LastPositionAt,
		Price:          t.Price,

		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},

		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		StartedAt: *t.StartedAt,
		Status:    ToAPIStatus(t.Status),
		UserId:    t.UserId,
	}
}

func ToAPIStatus(s model.Status) api.TripStatus {
	switch s {
	case model.Completed:
		return api.Completed
	default:
		return api.Active
	}
}
