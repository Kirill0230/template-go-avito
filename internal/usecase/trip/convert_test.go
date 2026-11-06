package trip_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/model"
	tripmapper "github.com/Kirill0230/template-go-avito/internal/usecase/trip"
)

func TestModelDomainRoundTrip(t *testing.T) {
	startedAt := time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)

	tests := []struct {
		name       string
		status     model.Status
		finishedAt *time.Time
	}{
		{
			name:   "active trip",
			status: model.Active,
		},
		{
			name:       "completed trip",
			status:     model.Completed,
			finishedAt: &finishedAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := model.Trip{
				Id:             uuid.New(),
				UserId:         uuid.New(),
				DriverId:       uuid.New(),
				Price:          1000,
				Status:         tt.status,
				StartedAt:      &startedAt,
				FinishedAt:     tt.finishedAt,
				LastPositionAt: &startedAt,
				StartPoint: model.Point{
					Latitude:  55.75,
					Longitude: 37.61,
				},
				EndPoint: model.Point{
					Latitude:  59.93,
					Longitude: 30.31,
				},
			}

			domainTrip, err := tripmapper.ModelToDomain(expected)
			if err != nil {
				t.Fatalf("ModelToDomain() error: %v", err)
			}

			actual := tripmapper.DomainToModel(domainTrip)

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf(
					"round trip result = %#v, want %#v",
					actual,
					expected,
				)
			}
		})
	}
}

func TestModelToDomainUnknownStatus(t *testing.T) {
	_, err := tripmapper.ModelToDomain(model.Trip{
		Status: model.Status("unknown"),
	})

	if err == nil {
		t.Fatal("ModelToDomain() error is nil")
	}
}
