package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Kirill0230/template-go-avito/internal/domain"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	pool         *pgxpool.Pool
	tripService  *service.TripService
	queryTimeout time.Duration
}

func NewServer(pool *pgxpool.Pool, tripService *service.TripService, queryTimeout time.Duration) *Server {
	return &Server{pool: pool, tripService: tripService, queryTimeout: queryTimeout}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	for _, name := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		if _, ok := fields[name]; !ok {
			writeError(w, r, http.StatusBadRequest, "invalid_request", name+" is required")
			return
		}
	}

	var body api.TripData
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err = validateTripData(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	trip := domain.Trip{
		UserId:   body.UserId,
		DriverId: body.DriverId,
		Price:    body.Price,
		StartPoint: domain.Point{
			Latitude:  body.StartPoint.Latitude,
			Longitude: body.StartPoint.Longitude,
		},
		EndPoint: domain.Point{
			Latitude:  body.EndPoint.Latitude,
			Longitude: body.EndPoint.Longitude,
		},
	}

	createdTrip, created, createdErr := s.tripService.CreateTrip(r.Context(), trip, params.IdempotencyKey)

	if createdErr != nil {
		switch {
		case errors.Is(createdErr, domain.ErrDriverBusy):
			writeError(w, r, http.StatusConflict, "driver_busy", createdErr.Error())
		case errors.Is(createdErr, domain.ErrIdempotencyConflict):
			writeError(w, r, http.StatusConflict, "idempotency_conflict", createdErr.Error())
		default:
			log.Printf("create trip: %v", createdErr)
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	if created {
		w.Header().Set("Location", "/api/v1/trips/"+createdTrip.Id.String())
		writeJSON(w, http.StatusCreated, toAPITrip(createdTrip))
		return
	}
	writeJSON(w, http.StatusOK, toAPITrip(createdTrip))
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := s.tripService.GetTrip(r.Context(), tripId)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTripNotFound):
			writeError(w, r, http.StatusNotFound, "trip_not_found", err.Error())
		default:
			log.Printf("get trip: %v", err)
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := s.tripService.FinishTrip(r.Context(), tripId)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTripNotFound):
			writeError(w, r, http.StatusNotFound, "trip_not_found", err.Error())
		case errors.Is(err, domain.ErrTripAlreadyComplete):
			writeError(w, r, http.StatusConflict, "trip_completed", err.Error())
		default:
			log.Printf("finish trip: %v", err)
			writeError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()

	err := s.pool.Ping(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    http.StatusText(status),
		Status:   int32(status),
		Code:     code,
		Detail:   &message,
		Instance: &instance,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func toAPITrip(t *domain.Trip) api.Trip {
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
		Status:    toAPIStatus(t.Status),
		UserId:    t.UserId,
	}
}

func toAPIStatus(s domain.Status) api.TripStatus {
	switch s {
	case domain.Completed:
		return api.Completed
	default:
		return api.Active
	}
}

func validateTripData(d api.TripData) error {
	if d.UserId == uuid.Nil {
		return errors.New("user_id is required")
	}
	if d.DriverId == uuid.Nil {
		return errors.New("driver_id is required")
	}
	err := validatePoint("start_point", d.StartPoint)
	if err != nil {
		return err
	}
	err = validatePoint("end_point", d.EndPoint)
	if err != nil {
		return err
	}
	if d.Price < 0 {
		return errors.New("price must be >= 0")
	}
	return nil
}

func validatePoint(name string, p api.Coordinates) error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return fmt.Errorf("%s.latitude out of range", name)
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return fmt.Errorf("%s.longitude out of range", name)
	}
	return nil
}
