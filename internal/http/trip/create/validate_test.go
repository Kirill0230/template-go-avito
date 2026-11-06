package create

import (
	"testing"

	"github.com/google/uuid"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

func TestValidateTripData(t *testing.T) {
	valid := api.TripData{
		UserId:     uuid.New(),
		DriverId:   uuid.New(),
		Price:      1000,
		StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   api.Coordinates{Latitude: 59.93, Longitude: 30.31},
	}

	tests := []struct {
		name    string
		change  func(*api.TripData)
		wantErr bool
	}{
		{
			name:    "valid",
			change:  func(*api.TripData) {},
			wantErr: false,
		},
		{
			name: "missing user",
			change: func(data *api.TripData) {
				data.UserId = uuid.Nil
			},
			wantErr: true,
		},
		{
			name: "negative price",
			change: func(data *api.TripData) {
				data.Price = -1
			},
			wantErr: true,
		},
		{
			name: "invalid latitude",
			change: func(data *api.TripData) {
				data.StartPoint.Latitude = 91
			},
			wantErr: true,
		},
		{
			name: "invalid longitude",
			change: func(data *api.TripData) {
				data.EndPoint.Longitude = 181
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := valid
			tt.change(&data)

			err := validateTripData(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateTripData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
