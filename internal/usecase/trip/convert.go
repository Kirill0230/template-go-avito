package trip

import (
	"fmt"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

func ModelToDomain(modelTrip model.Trip) (*domaintrip.Trip, error) {
	status, err := modelStatusToDomain(modelTrip.Status)
	if err != nil {
		return nil, err
	}

	return domaintrip.RestoreTrip(
		modelTrip.Id,
		modelTrip.UserId,
		modelTrip.DriverId,
		modelTrip.Price,
		status,
		domaintrip.Point{
			Latitude:  modelTrip.StartPoint.Latitude,
			Longitude: modelTrip.StartPoint.Longitude,
		},
		domaintrip.Point{
			Latitude:  modelTrip.EndPoint.Latitude,
			Longitude: modelTrip.EndPoint.Longitude,
		},
		modelTrip.StartedAt,
		modelTrip.FinishedAt,
		modelTrip.LastPositionAt,
	), nil
}

func DomainToModel(domainTrip *domaintrip.Trip) model.Trip {
	return model.Trip{
		Id:       domainTrip.Id(),
		UserId:   domainTrip.UserId(),
		DriverId: domainTrip.DriverId(),
		Price:    domainTrip.Price(),
		Status:   domainStatusToModel(domainTrip.Status()),
		StartPoint: model.Point{
			Latitude:  domainTrip.StartPoint().Latitude,
			Longitude: domainTrip.StartPoint().Longitude,
		},
		EndPoint: model.Point{
			Latitude:  domainTrip.EndPoint().Latitude,
			Longitude: domainTrip.EndPoint().Longitude,
		},
		StartedAt:      domainTrip.StartedAt(),
		FinishedAt:     domainTrip.FinishedAt(),
		LastPositionAt: domainTrip.LastPositionAt(),
	}
}

func modelStatusToDomain(status model.Status) (domaintrip.Status, error) {
	switch status {
	case model.Active:
		return domaintrip.Active, nil
	case model.Completed:
		return domaintrip.Completed, nil
	default:
		return "", fmt.Errorf("unknown trip status: %q", status)
	}
}

func domainStatusToModel(status domaintrip.Status) model.Status {
	switch status {
	case domaintrip.Active:
		return model.Active
	case domaintrip.Completed:
		return model.Completed
	default:
		panic(fmt.Sprintf("unknown domain trip status: %q", status))
	}
}
