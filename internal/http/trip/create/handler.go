package create

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	tripapi "github.com/Kirill0230/template-go-avito/internal/http/trip"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

const maxRequestBodyBytes = 1 << 20

type Handler struct {
	usecase usecase
	logger  *slog.Logger
}

func NewHandler(
	usecase usecase,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err = checkRequired(raw)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var body api.TripData
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&body)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err = validateTripData(body)
	if err != nil {
		response.WriteError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	trip := model.Trip{
		UserId:   body.UserId,
		DriverId: body.DriverId,
		Price:    body.Price,
		StartPoint: model.Point{
			Latitude:  body.StartPoint.Latitude,
			Longitude: body.StartPoint.Longitude,
		},
		EndPoint: model.Point{
			Latitude:  body.EndPoint.Latitude,
			Longitude: body.EndPoint.Longitude,
		},
	}

	createdTrip, created, createdErr := h.usecase.CreateTrip(r.Context(), trip, params.IdempotencyKey)

	if createdErr != nil {
		status, problem := domainerror.Problem(createdErr)
		response.WriteProblem(w, r, status, problem)
		return
	}

	if created {
		attributes := []any{
			slog.String(
				"trip_id",
				createdTrip.Id.String(),
			),
		}

		spanContext := trace.SpanContextFromContext(
			r.Context(),
		)

		if spanContext.IsValid() {
			attributes = append(
				attributes,
				slog.String(
					"trace_id",
					spanContext.TraceID().String(),
				),
				slog.String(
					"span_id",
					spanContext.SpanID().String(),
				),
			)
		}

		h.logger.InfoContext(
			r.Context(),
			"trip created",
			attributes...,
		)
	}

	if created {
		w.Header().Set("Location", "/api/v1/trips/"+createdTrip.Id.String())
		response.WriteJSON(w, http.StatusCreated, tripapi.ToAPITrip(createdTrip))
		return
	}
	response.WriteJSON(w, http.StatusOK, tripapi.ToAPITrip(createdTrip))
}
