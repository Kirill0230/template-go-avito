package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(next http.Handler) http.Handler {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			route := chi.RouteContext(
				r.Context(),
			).RoutePattern()

			if route == "" {
				return
			}

			span := trace.SpanFromContext(r.Context())

			span.SetName(r.Method + " " + route)

			span.SetAttributes(
				attribute.String("http.route", route),
			)
		},
	)

	return otelhttp.NewHandler(
		handler,
		"HTTP request",
		// HTTP metrics are emitted by Metrics. Keeping otelhttp metrics
		// enabled would record http.server.request.duration twice.
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
	)
}
