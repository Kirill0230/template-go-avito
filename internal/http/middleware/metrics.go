package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
}

func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription(
			"HTTP request duration",
		),
	)
	if err != nil {
		return nil, err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription(
			"Number of active HTTP requests",
		),
	)
	if err != nil {
		return nil, err
	}

	return &HTTPMetrics{
		requestDuration: requestDuration,
		activeRequests:  activeRequests,
	}, nil
}

func Metrics(metrics *HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				startedAt := time.Now()

				route := chi.RouteContext(
					r.Context(),
				).RoutePattern()

				methodAttribute := attribute.String(
					"http.request.method",
					r.Method,
				)

				routeAttribute := attribute.String(
					"http.route",
					route,
				)

				metrics.activeRequests.Add(
					r.Context(),
					1,
					metric.WithAttributes(
						methodAttribute,
						routeAttribute,
					),
				)

				defer metrics.activeRequests.Add(
					context.WithoutCancel(r.Context()),
					-1,
					metric.WithAttributes(
						methodAttribute,
						routeAttribute,
					),
				)

				writer := &responseWriter{
					ResponseWriter: w,
					statusCode:     http.StatusOK,
				}

				next.ServeHTTP(writer, r)

				metrics.requestDuration.Record(
					r.Context(),
					time.Since(startedAt).Seconds(),
					metric.WithAttributes(
						methodAttribute,
						routeAttribute,
						attribute.Int(
							"http.response.status_code",
							writer.statusCode,
						),
					),
				)
			},
		)
	}
}
