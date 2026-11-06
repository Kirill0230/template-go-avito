package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

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

					logger.ErrorContext(
						r.Context(),
						"panic recovered",
						slog.Any("panic", recovered),
						slog.String(
							"stack",
							string(debug.Stack()),
						),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
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
