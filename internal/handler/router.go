package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Kirill0230/template-go-avito/internal/config"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

func (s *Server) NewRouter(ctx context.Context, cfg config.HTTPConfig) error {
	router := api.HandlerWithOptions(s, api.ChiServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		},
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		_ = srv.Close()
		return fmt.Errorf("shutdown timeout exceeded, forced close: %w", err)
	}
	return nil
}
