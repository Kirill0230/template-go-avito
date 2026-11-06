package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter

	statusCode  int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode
	w.wroteHeader = true

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()

			writer := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(writer, r)

			duration := time.Since(startedAt)
			route := chi.RouteContext(r.Context()).RoutePattern()

			attributes := []any{
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", writer.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
			}

			spanContext := trace.SpanContextFromContext(r.Context())

			if spanContext.IsValid() {
				attributes = append(
					attributes,
					slog.String(
						"trace_id",
						spanContext.TraceID().String(),
					),
					slog.String(
						"span_id",
						spanContext.SpanID().String(),
					),
				)
			}

			if writer.statusCode >= http.StatusInternalServerError {
				logger.ErrorContext(
					r.Context(),
					"request handled",
					attributes...,
				)

				return
			}

			logger.InfoContext(
				r.Context(),
				"request handled",
				attributes...,
			)
		})
	}
}
