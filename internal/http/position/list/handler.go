package list

import (
	"net/http"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	"github.com/Kirill0230/template-go-avito/internal/model"
)

type Handler struct{ usecase usecase }

func NewHandler(usecase usecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	positions, err := h.usecase.List(r.Context(), tripID)
	if err != nil {
		status, problem := domainerror.Problem(err)
		response.WriteProblem(w, r, status, problem, err)
		return
	}

	result := make([]api.TripPosition, 0, len(positions))
	for _, position := range positions {
		result = append(result, toAPIPosition(position))
	}
	response.WriteJSON(w, http.StatusOK, api.TripPositionsResponse{Positions: result})
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
