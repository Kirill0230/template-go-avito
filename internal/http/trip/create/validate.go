package create

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

const (
	startPointField = "start_point"
	endPointField   = "end_point"
)

func validatePoint(name string, p api.Coordinates) error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return fmt.Errorf("%s.latitude out of range", name)
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return fmt.Errorf("%s.longitude out of range", name)
	}
	return nil
}

func checkRequired(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return errors.New("request body must be a JSON object")
	}

	for name := range fields {
		switch name {
		case "user_id", "driver_id", startPointField, endPointField, "price":
		default:
			return fmt.Errorf("unknown field %q", name)
		}
	}

	for _, name := range []string{"user_id", "driver_id", startPointField, endPointField, "price"} {
		v, ok := fields[name]
		if !ok || string(v) == "null" {
			return fmt.Errorf("%s is required", name)
		}
	}
	for _, p := range []string{startPointField, endPointField} {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(fields[p], &point); err != nil || point == nil {
			return fmt.Errorf("%s must be an object", p)
		}

		for name := range point {
			if name != "latitude" && name != "longitude" {
				return fmt.Errorf("unknown field %q", p+"."+name)
			}
		}

		for _, c := range []string{"latitude", "longitude"} {
			v, ok := point[c]
			if !ok || string(v) == "null" {
				return fmt.Errorf("%s.%s is required", p, c)
			}
		}
	}
	return nil
}

func validateTripData(d api.TripData) error {
	if d.UserId == uuid.Nil {
		return errors.New("user_id is required")
	}
	if d.DriverId == uuid.Nil {
		return errors.New("driver_id is required")
	}
	err := validatePoint(startPointField, d.StartPoint)
	if err != nil {
		return err
	}
	err = validatePoint(endPointField, d.EndPoint)
	if err != nil {
		return err
	}
	if d.Price < 0 {
		return errors.New("price must be >= 0")
	}
	return nil
}
