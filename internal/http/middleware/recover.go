package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"

	"github.com/Kirill0230/template-go-avito/internal/http/response"
)

func Recover(
	logger *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				writer := &responseWriter{
					ResponseWriter: w,
					statusCode:     http.StatusOK,
				}

				defer func() {
					recovered := recover()
					if recovered == nil {
						return
					}

					attributes := []any{
						slog.Any("panic", recovered),
						slog.String(
							"stack",
							string(debug.Stack()),
						),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					}

					spanContext := trace.SpanContextFromContext(r.Context())
					if spanContext.IsValid() {
						attributes = append(
							attributes,
							slog.String("trace_id", spanContext.TraceID().String()),
							slog.String("span_id", spanContext.SpanID().String()),
						)
					}

					logger.ErrorContext(
						r.Context(),
						"panic recovered",
						attributes...,
					)

					if writer.wroteHeader {
						return
					}

					response.WriteError(
						writer,
						r,
						http.StatusInternalServerError,
						"internal_error",
						"Internal server error",
					)
				}()

				next.ServeHTTP(writer, r)
			},
		)
	}
}
