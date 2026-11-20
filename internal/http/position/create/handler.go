package create

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

const maxRequestBodyBytes = 1 << 20

type Handler struct{ usecase usecase }

func NewHandler(usecase usecase) *Handler { return &Handler{usecase: usecase} }

type coordinatesRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type positionRequest struct {
	Coordinates *coordinatesRequest `json:"coordinates"`
	RecordedAt  *time.Time          `json:"recorded_at"`
}

func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var body positionRequest
	if err := decoder.Decode(&body); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if body.Coordinates == nil || body.Coordinates.Latitude == nil ||
		body.Coordinates.Longitude == nil || body.RecordedAt == nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", "coordinates and recorded_at are required")
		return
	}

	position, err := h.usecase.Create(r.Context(), model.TripPosition{
		TripID: tripID,
		Coordinates: model.Point{
			Latitude:  *body.Coordinates.Latitude,
			Longitude: *body.Coordinates.Longitude,
		},
		RecordedAt: *body.RecordedAt,
	})
	if err != nil {
		if errors.Is(err, domaintrip.ErrInvalidCoordinates) || errors.Is(err, domaintrip.ErrInvalidRecordedAt) {
			response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		status, problem := domainerror.Problem(err)
		response.WriteProblem(w, r, status, problem, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, toAPIPosition(*position))
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return err
	}
	return nil
}

func toAPIPosition(position model.TripPosition) api.TripPosition {
	return api.TripPosition{
		Id: position.ID, TripId: position.TripID,
		Coordinates: api.Coordinates{
			Latitude: position.Coordinates.Latitude, Longitude: position.Coordinates.Longitude,
		},
		RecordedAt: position.RecordedAt,
	}
}
