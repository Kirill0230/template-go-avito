package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Kirill0230/template-go-avito/internal/config"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

func (s *Server) NewRouter(ctx context.Context, cfg config.HTTPConfig) {
	router := api.HandlerWithOptions(s, api.ChiServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		},
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Println("shutdown error:", err)
	}
}
