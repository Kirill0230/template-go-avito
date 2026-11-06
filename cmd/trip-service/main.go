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

func run() (resultErr error) {
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
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.HTTP.ShutdownTimeout,
		)
		defer cancel()

		resultErr = errors.Join(
			resultErr,
			app.Shutdown(shutdownCtx),
		)
	}()

	app.Logger.InfoContext(
		ctx,
		"HTTP server starting",
		"address",
		cfg.HTTP.Addr,
	)

	if err := httpserver.Run(
		ctx,
		app.Handler,
		cfg.HTTP,
	); err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	app.Logger.Info(
		"HTTP server stopped",
	)

	return nil
}
