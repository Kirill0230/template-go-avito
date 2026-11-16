package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/Kirill0230/template-go-avito/internal/config"
	httpserver "github.com/Kirill0230/template-go-avito/internal/http"
	"github.com/Kirill0230/template-go-avito/internal/http/health"
	httpmiddleware "github.com/Kirill0230/template-go-avito/internal/http/middleware"
	createhandler "github.com/Kirill0230/template-go-avito/internal/http/trip/create"
	finishhandler "github.com/Kirill0230/template-go-avito/internal/http/trip/finish"
	gethandler "github.com/Kirill0230/template-go-avito/internal/http/trip/get"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres"
	"github.com/Kirill0230/template-go-avito/internal/infra/postgres/transaction"
	"github.com/Kirill0230/template-go-avito/internal/infra/telemetry"
	"github.com/Kirill0230/template-go-avito/internal/repository/idempotency"
	triprepository "github.com/Kirill0230/template-go-avito/internal/repository/trip"
	statushistory "github.com/Kirill0230/template-go-avito/internal/repository/trip_status_history"
	createusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/create"
	finishusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/finish"
	getusecase "github.com/Kirill0230/template-go-avito/internal/usecase/trip/get"
)

const serviceName = "trip-service"

type Container struct {
	Handler http.Handler
	Logger  *slog.Logger

	pool           *pgxpool.Pool
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger, err := telemetry.NewLogger(
		serviceName,
		cfg.LogLevel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create logger: %w",
			err,
		)
	}

	tracerProvider, err := telemetry.NewTracerProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create tracer provider: %w",
			err,
		)
	}

	meterProvider, err := telemetry.NewMeterProvider(ctx)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)

		return nil, fmt.Errorf(
			"create meter provider: %w",
			err,
		)
	}

	meter := meterProvider.Meter(
		"github.com/Kirill0230/template-go-avito/internal/http/middleware",
	)

	httpMetrics, err := httpmiddleware.NewHTTPMetrics(meter)
	if err != nil {
		_ = meterProvider.Shutdown(ctx)
		_ = tracerProvider.Shutdown(ctx)

		return nil, fmt.Errorf(
			"create HTTP metrics: %w",
			err,
		)
	}

	tripMetrics, err := telemetry.NewTripMetrics(meter)
	if err != nil {
		_ = meterProvider.Shutdown(ctx)
		_ = tracerProvider.Shutdown(ctx)

		return nil, fmt.Errorf(
			"create trip metrics: %w",
			err,
		)
	}

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		_ = meterProvider.Shutdown(ctx)
		_ = tracerProvider.Shutdown(ctx)

		return nil, fmt.Errorf(
			"create postgres pool: %w",
			err,
		)
	}

	tripRepo := triprepository.NewTripRepository(pool, cfg.DatabaseQueryTimeout)

	tripStatusHistoryRepo := statushistory.NewTripStatusHistoryRepository(pool, cfg.DatabaseQueryTimeout)

	idempotencyRepo := idempotency.NewIdempotencyKeyRepository(pool, cfg.DatabaseQueryTimeout)

	txManager := transaction.NewTxManagerImpl(pool, cfg.DatabaseQueryTimeout)

	createService := createusecase.NewService(tripRepo, tripStatusHistoryRepo, idempotencyRepo, txManager, tripMetrics)

	getService := getusecase.NewService(tripRepo)

	finishService := finishusecase.NewService(tripRepo, tripStatusHistoryRepo, txManager, tripMetrics)

	createHandler := createhandler.NewHandler(createService, logger)

	getHandler := gethandler.NewHandler(getService)

	finishHandler := finishhandler.NewHandler(finishService, logger)

	healthHandler := health.NewHandler(pool, cfg.DatabaseQueryTimeout)

	server := httpserver.NewServer(createHandler, getHandler, finishHandler, healthHandler)

	handler := httpserver.NewRouter(server, logger, httpMetrics)

	return &Container{
		Handler:        handler,
		Logger:         logger,
		pool:           pool,
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}, nil
}

func (c *Container) Shutdown(ctx context.Context) error {
	c.pool.Close()

	return errors.Join(
		c.meterProvider.Shutdown(ctx),
		c.tracerProvider.Shutdown(ctx),
	)
}
