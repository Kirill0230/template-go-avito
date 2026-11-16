package create_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	createhttp "github.com/Kirill0230/template-go-avito/internal/http/trip/create"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func validRequestBody() string {
	return `{
		"user_id":"00000000-0000-0000-0000-000000000001",
		"driver_id":"00000000-0000-0000-0000-000000000002",
		"start_point":{"latitude":1,"longitude":2},
		"end_point":{"latitude":3,"longitude":4},
		"price":100
	}`
}

func TestCreateHandlerInvalidRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	usecase := NewMockusecase(ctrl)

	handler := createhttp.NewHandler(usecase, discardLogger())

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(`{"unknown":true}`),
	)
	response := httptest.NewRecorder()

	handler.CreateTrip(response, request, api.CreateTripParams{})

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var problem api.Problem
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if problem.Code != "invalid_request" {
		t.Fatalf("code = %q, want invalid_request", problem.Code)
	}
}

func TestCreateHandlerInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	usecase := NewMockusecase(ctrl)

	internalMessage := "database password leaked"

	usecase.EXPECT().
		CreateTrip(
			gomock.Any(),
			gomock.Any(),
			(*uuid.UUID)(nil),
		).
		Return(nil, false, errors.New(internalMessage))

	handler := createhttp.NewHandler(usecase, discardLogger())

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(validRequestBody()),
	)
	response := httptest.NewRecorder()

	handler.CreateTrip(response, request, api.CreateTripParams{})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	if strings.Contains(response.Body.String(), internalMessage) {
		t.Fatal("internal error details leaked into response")
	}
}
