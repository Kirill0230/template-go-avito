package finish_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/Kirill0230/template-go-avito/internal/domain/trip"
	api "github.com/Kirill0230/template-go-avito/internal/generated"
	finishhttp "github.com/Kirill0230/template-go-avito/internal/http/trip/finish"
)

func TestFinishHandlerCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	usecase := NewMockusecase(ctrl)

	tripID := uuid.New()

	usecase.EXPECT().
		FinishTrip(gomock.Any(), tripID).
		Return(nil, domaintrip.ErrTripCompleted)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := finishhttp.NewHandler(usecase, logger)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+tripID.String()+"/finish",
		nil,
	)
	response := httptest.NewRecorder()

	handler.FinishTrip(response, request, tripID)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}

	var problem api.Problem
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if problem.Code != "trip_completed" {
		t.Fatalf("code = %q, want trip_completed", problem.Code)
	}
}
