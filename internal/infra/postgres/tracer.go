package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type collectionContextKey struct{}

type queryMetricsContextKey struct{}

type queryMetrics struct {
	startedAt  time.Time
	operation  string
	collection string
}

type tracer struct {
	*otelpgx.Tracer
	operationDuration metric.Float64Histogram
}

func newTracer() (*tracer, error) {
	duration, err := otel.Meter(
		"github.com/Kirill0230/template-go-avito/internal/infra/postgres",
	).Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of database client operations"),
		metric.WithExplicitBucketBoundaries(
			0.001,
			0.005,
			0.01,
			0.05,
			0.1,
			0.5,
			1,
			5,
			10,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create database operation duration metric: %w", err)
	}

	return &tracer{
		// The wrapper below emits db.client.operation.duration with the
		// required per-query attributes. Disable the same otelpgx metric to
		// avoid recording each database operation twice.
		Tracer: otelpgx.NewTracer(
			otelpgx.WithMeterProvider(metricnoop.NewMeterProvider()),
		),
		operationDuration: duration,
	}, nil
}

func WithDBCollection(ctx context.Context, collection string) context.Context {
	return context.WithValue(ctx, collectionContextKey{}, collection)
}

func (t *tracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	startedAt := time.Now()
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)

	collection, _ := ctx.Value(collectionContextKey{}).(string)
	operation := otelpgx.SQLOperationName(ctx, data.SQL)
	if collection != "" {
		trace.SpanFromContext(ctx).SetAttributes(semconv.DBCollectionName(collection))
	}

	return context.WithValue(ctx, queryMetricsContextKey{}, queryMetrics{
		startedAt:  startedAt,
		operation:  operation,
		collection: collection,
	})
}

func (t *tracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	query, ok := ctx.Value(queryMetricsContextKey{}).(queryMetrics)
	if ok {
		attributes := []attribute.KeyValue{
			semconv.DBOperationName(query.operation),
		}
		if query.collection != "" {
			attributes = append(attributes, semconv.DBCollectionName(query.collection))
		}
		if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
			errorType := semconv.ErrorType(data.Err)
			attributes = append(attributes, errorType)
			trace.SpanFromContext(ctx).SetAttributes(errorType)
		}

		t.operationDuration.Record(
			ctx,
			time.Since(query.startedAt).Seconds(),
			metric.WithAttributes(attributes...),
		)
	}

	t.Tracer.TraceQueryEnd(ctx, conn, data)
}
