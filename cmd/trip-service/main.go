package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/Kirill0230/template-go-avito/internal/config"
	"github.com/Kirill0230/template-go-avito/internal/container"
	httpserver "github.com/Kirill0230/template-go-avito/internal/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	app, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	server := httpserver.NewHTTPServer(app.Handler, cfg.HTTP)

	app.Logger.InfoContext(
		ctx,
		"HTTP server starting",
		"address",
		cfg.HTTP.Addr,
	)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run()
	}()

	var runErr error
	serverStopped := false
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		serverStopped = true
		if err != nil {
			runErr = fmt.Errorf("run HTTP server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		cfg.HTTP.ShutdownTimeout,
	)
	defer cancel()

	httpShutdownErr := server.Shutdown(shutdownCtx)
	if !serverStopped {
		if err := <-serverErr; err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("run HTTP server: %w", err))
		}
	}
	containerShutdownErr := app.Shutdown(shutdownCtx)

	app.Logger.Info(
		"HTTP server stopped",
	)

	return errors.Join(
		runErr,
		httpShutdownErr,
		containerShutdownErr,
	)
}
