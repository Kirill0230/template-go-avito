package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/health"
	httpmiddleware "github.com/Kirill0230/template-go-avito/internal/http/middleware"
	positioncreate "github.com/Kirill0230/template-go-avito/internal/http/position/create"
	positionlist "github.com/Kirill0230/template-go-avito/internal/http/position/list"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
	"github.com/Kirill0230/template-go-avito/internal/http/trip/create"
	"github.com/Kirill0230/template-go-avito/internal/http/trip/finish"
	"github.com/Kirill0230/template-go-avito/internal/http/trip/get"
)

type Server struct {
	create         *create.Handler
	get            *get.Handler
	finish         *finish.Handler
	health         *health.Handler
	positionCreate *positioncreate.Handler
	positionList   *positionlist.Handler
}

func NewServer(
	createH *create.Handler,
	getH *get.Handler,
	finishH *finish.Handler,
	healthH *health.Handler,
	positionCreateH *positioncreate.Handler,
	positionListH *positionlist.Handler,
) *Server {
	return &Server{
		create:         createH,
		get:            getH,
		finish:         finishH,
		health:         healthH,
		positionCreate: positionCreateH,
		positionList:   positionListH,
	}
}

func NewRouter(
	s *Server,
	logger *slog.Logger,
	httpMetrics *httpmiddleware.HTTPMetrics,
) http.Handler {
	router := chi.NewRouter()

	router.Use(
		httpmiddleware.Tracing,
		httpmiddleware.Logging(logger),
		httpmiddleware.Metrics(httpMetrics),
		httpmiddleware.Recover(logger),
	)

	return api.HandlerWithOptions(
		s,
		api.ChiServerOptions{
			BaseRouter: router,
			ErrorHandlerFunc: func(
				w http.ResponseWriter,
				r *http.Request,
				err error,
			) {
				response.WriteError(
					w,
					r,
					http.StatusBadRequest,
					"invalid_request",
					err.Error(),
				)
			},
		},
	)
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.create.CreateTrip(w, r, params)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.get.GetTrip(w, r, tripId)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.finish.FinishTrip(w, r, tripId)
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	s.health.Health(w, r)
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	s.health.Ready(w, r)
}

func (s *Server) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.positionCreate.CreateTripPosition(w, r, tripID)
}

func (s *Server) ListTripPositions(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.positionList.ListTripPositions(w, r, tripID)
}
