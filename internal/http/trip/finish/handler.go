package finish

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	tripapi "github.com/Kirill0230/template-go-avito/internal/http/trip"
)

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

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.usecase.FinishTrip(r.Context(), tripId)
	if err != nil {
		status, problem := domainerror.Problem(err)
		response.WriteProblem(w, r, status, problem)
		return
	}

	attributes := []any{
		slog.String(
			"trip_id",
			trip.Id.String(),
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
		"trip completed",
		attributes...,
	)

	response.WriteJSON(w, http.StatusOK, tripapi.ToAPITrip(trip))
}
