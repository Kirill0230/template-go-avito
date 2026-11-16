package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Kirill0230/template-go-avito/internal/config"
)

type HTTPServer struct {
	server *http.Server
}

func NewHTTPServer(handler http.Handler, cfg config.HTTPConfig) *HTTPServer {
	return &HTTPServer{server: &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}}
}

func (s *HTTPServer) Run() error {
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}

	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		_ = s.server.Close()
		return fmt.Errorf("shutdown timeout exceeded, forced close: %w", err)
	}

	return nil
}
