package get

import (
	"net/http"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	tripapi "github.com/Kirill0230/template-go-avito/internal/http/trip"
)

type Handler struct {
	usecase usecase
}

func NewHandler(usecase usecase) *Handler {
	return &Handler{
		usecase: usecase,
	}
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.usecase.GetTrip(r.Context(), tripId)
	if err != nil {
		status, problem := domainerror.Problem(err)
		response.WriteProblem(w, r, status, problem)
		return
	}

	response.WriteJSON(w, http.StatusOK, tripapi.ToAPITrip(trip))
}
