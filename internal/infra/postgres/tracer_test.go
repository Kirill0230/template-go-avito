package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestTracerAddsDatabaseAttributes(t *testing.T) {
	ctx := context.Background()

	previousMeterProvider := otel.GetMeterProvider()
	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	otel.SetMeterProvider(meterProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		if err := meterProvider.Shutdown(ctx); err != nil {
			t.Errorf("shutdown meter provider: %v", err)
		}
	})

	previousTracerProvider := otel.GetTracerProvider()
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(tracerProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		if err := tracerProvider.Shutdown(ctx); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
	})

	dbTracer, err := newTracer()
	if err != nil {
		t.Fatalf("newTracer() error: %v", err)
	}

	ctx, parentSpan := tracerProvider.Tracer("test").Start(ctx, "request")
	defer parentSpan.End()

	queryContext := WithDBCollection(ctx, "trips")
	queryContext = dbTracer.TraceQueryStart(queryContext, nil, pgx.TraceQueryStartData{
		SQL: "SELECT id FROM trips WHERE id = $1",
	})
	queryErr := errors.New("database unavailable")
	dbTracer.TraceQueryEnd(queryContext, nil, pgx.TraceQueryEndData{Err: queryErr})

	wantErrorType := semconv.ErrorType(queryErr).Value.AsString()
	assertDatabaseMetricAttributes(t, ctx, metricReader, wantErrorType)
	assertDatabaseSpanAttributes(t, spanRecorder.Ended(), wantErrorType)
}

func assertDatabaseMetricAttributes(
	t *testing.T,
	ctx context.Context,
	reader *sdkmetric.ManualReader,
	wantErrorType string,
) {
	t.Helper()

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	for _, scope := range collected.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "db.client.operation.duration" {
				continue
			}
			histogram, ok := candidate.Data.(metricdata.Histogram[float64])
			if !ok || len(histogram.DataPoints) != 1 {
				t.Fatalf("database duration metric has unexpected data: %#v", candidate.Data)
			}
			assertAttribute(t, histogram.DataPoints[0].Attributes, "db.operation.name", "SELECT")
			assertAttribute(t, histogram.DataPoints[0].Attributes, "db.collection.name", "trips")
			assertAttribute(t, histogram.DataPoints[0].Attributes, "error.type", wantErrorType)
			return
		}
	}

	t.Fatal("db.client.operation.duration metric not found")
}

func assertDatabaseSpanAttributes(
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	wantErrorType string,
) {
	t.Helper()

	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}

	attributes := attribute.NewSet(spans[0].Attributes()...)
	assertAttribute(t, attributes, "db.operation.name", "SELECT")
	assertAttribute(t, attributes, "db.collection.name", "trips")
	assertAttribute(t, attributes, "error.type", wantErrorType)
}

func assertAttribute(t *testing.T, attributes attribute.Set, key, want string) {
	t.Helper()

	value, ok := attributes.Value(attribute.Key(key))
	if !ok || value.AsString() != want {
		t.Fatalf("attribute %q = %q, want %q", key, value.AsString(), want)
	}
}
