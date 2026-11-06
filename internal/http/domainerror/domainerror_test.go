package domainerror_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	"github.com/Kirill0230/template-go-avito/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "trip not found",
			err:        domaintrip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "trip completed",
			err:        domaintrip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "driver busy",
			err:        domaintrip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "idempotency conflict",
			err:        domaintrip.ErrIdempotencyConflict,
			wantStatus: http.StatusConflict,
			wantCode:   "idempotency_conflict",
		},
		{
			name:       "internal error",
			err:        errors.New("database password leaked"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, problem := domainerror.Problem(tt.err)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			if problem.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", problem.Code, tt.wantCode)
			}

			if tt.wantStatus == http.StatusInternalServerError &&
				problem.Detail != nil &&
				strings.Contains(*problem.Detail, "database password") {
				t.Fatal("internal error details leaked into response")
			}
		})
	}
}
