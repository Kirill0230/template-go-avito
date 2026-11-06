package get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
	gethttp "github.com/Kirill0230/template-go-avito/internal/http/trip/get"
)

func TestGetHandlerNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	usecase := NewMockusecase(ctrl)

	tripID := uuid.New()

	usecase.EXPECT().
		GetTrip(gomock.Any(), tripID).
		Return(nil, domaintrip.ErrTripNotFound)

	handler := gethttp.NewHandler(usecase)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String(),
		nil,
	)
	response := httptest.NewRecorder()

	handler.GetTrip(response, request, tripID)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}

	var problem api.Problem
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if problem.Code != "trip_not_found" {
		t.Fatalf("code = %q, want trip_not_found", problem.Code)
	}
}
