package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	sampler, err := newSampler(os.Getenv("OTEL_TRACES_SAMPLER"))
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return provider, nil
}

func newSampler(value string) (sdktrace.Sampler, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "parentbased_always_on":
		return sdktrace.ParentBased(
			sdktrace.AlwaysSample(),
		), nil

	case "always_on":
		return sdktrace.AlwaysSample(), nil

	case "always_off":
		return sdktrace.NeverSample(), nil

	case "parentbased_always_off":
		return sdktrace.ParentBased(
			sdktrace.NeverSample(),
		), nil

	default:
		return nil, fmt.Errorf(
			"unsupported OTEL_TRACES_SAMPLER %q",
			value,
		)
	}
}
