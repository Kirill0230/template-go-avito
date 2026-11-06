package health

import (
	"context"
	"net/http"
	"time"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
)

type Handler struct {
	db      pinger
	timeout time.Duration
}

func NewHandler(db pinger, timeout time.Duration) *Handler {
	return &Handler{db: db, timeout: timeout}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	err := h.db.Ping(ctx)
	if err != nil {
		response.WriteJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	response.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
