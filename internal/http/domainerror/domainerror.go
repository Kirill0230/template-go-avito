package domainerror

import (
	"errors"
	"net/http"
	"strings"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	switch {
	case errors.Is(err, domaintrip.ErrTripNotFound):
		return newProblem(
			http.StatusNotFound,
			"trip_not_found",
			err.Error(),
		)

	case errors.Is(err, domaintrip.ErrTripCompleted):
		return newProblem(
			http.StatusConflict,
			"trip_completed",
			err.Error(),
		)

	case errors.Is(err, domaintrip.ErrDriverBusy):
		return newProblem(
			http.StatusConflict,
			"driver_busy",
			err.Error(),
		)

	case errors.Is(err, domaintrip.ErrIdempotencyConflict):
		return newProblem(
			http.StatusConflict,
			"idempotency_conflict",
			err.Error(),
		)

	default:
		return newProblem(
			http.StatusInternalServerError,
			"internal_error",
			"Internal server error",
		)
	}
}

func newProblem(status int32, code, detail string) (int, api.Problem) {
	return int(status), api.Problem{
		Type: "https://tripgo.example/problems/" +
			strings.ReplaceAll(code, "_", "-"),
		Title:  http.StatusText(int(status)),
		Status: status,
		Code:   code,
		Detail: &detail,
	}
}
