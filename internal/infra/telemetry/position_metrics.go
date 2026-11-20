package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionMetrics struct {
	staleChecks     metric.Int64Counter
	queueSize       metric.Int64UpDownCounter
	activeWorkers   metric.Int64UpDownCounter
	requests        metric.Int64Counter
	requestDuration metric.Float64Histogram
	saved           metric.Int64Counter
	rowsRead        metric.Int64Histogram
}

func NewPositionMetrics(meter metric.Meter) (*PositionMetrics, error) {
	staleChecks, err := meter.Int64Counter("tripgo.stale_checks")
	if err != nil {
		return nil, fmt.Errorf("create stale checks metric: %w", err)
	}
	queueSize, err := meter.Int64UpDownCounter("tripgo.position_requests.queue.size")
	if err != nil {
		return nil, fmt.Errorf("create position queue size metric: %w", err)
	}
	activeWorkers, err := meter.Int64UpDownCounter("tripgo.position_requests.workers.active")
	if err != nil {
		return nil, fmt.Errorf("create active position workers metric: %w", err)
	}
	requests, err := meter.Int64Counter("tripgo.position_requests")
	if err != nil {
		return nil, fmt.Errorf("create position requests metric: %w", err)
	}
	requestDuration, err := meter.Float64Histogram(
		"tripgo.position_requests.duration",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1),
	)
	if err != nil {
		return nil, fmt.Errorf("create position request duration metric: %w", err)
	}
	saved, err := meter.Int64Counter("tripgo.positions.saved")
	if err != nil {
		return nil, fmt.Errorf("create positions saved metric: %w", err)
	}
	rowsRead, err := meter.Int64Histogram(
		"tripgo.stale_checks.rows_read",
		metric.WithExplicitBucketBoundaries(10, 50, 100, 250, 500, 1000),
	)
	if err != nil {
		return nil, fmt.Errorf("create stale rows read metric: %w", err)
	}

	return &PositionMetrics{
		staleChecks: staleChecks, queueSize: queueSize,
		activeWorkers: activeWorkers, requests: requests,
		requestDuration: requestDuration, saved: saved, rowsRead: rowsRead,
	}, nil
}

func (m *PositionMetrics) StaleCheck(ctx context.Context, result string) {
	m.staleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) QueueChanged(ctx context.Context, delta int64) {
	m.queueSize.Add(ctx, delta)
}

func (m *PositionMetrics) ActiveWorkersChanged(ctx context.Context, delta int64) {
	m.activeWorkers.Add(ctx, delta)
}

func (m *PositionMetrics) PositionRequest(ctx context.Context, result string, duration time.Duration) {
	attributes := metric.WithAttributes(attribute.String("result", result))
	m.requests.Add(ctx, 1, attributes)
	m.requestDuration.Record(ctx, duration.Seconds(), attributes)
}

func (m *PositionMetrics) PositionSaved(ctx context.Context) {
	m.saved.Add(ctx, 1)
}

func (m *PositionMetrics) RowsRead(ctx context.Context, rows int64) {
	m.rowsRead.Record(ctx, rows)
}
