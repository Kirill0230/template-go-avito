package health_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Kirill0230/template-go-avito/internal/http/health"
	"github.com/Kirill0230/template-go-avito/internal/http/middleware"
)

func TestReadyLogsPingError(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := NewMockpinger(ctrl)
	pingErr := errors.New("connection refused")
	db.EXPECT().Ping(gomock.Any()).Return(pingErr)

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := health.NewHandler(db, time.Second)
	wrapped := middleware.Logging(logger)(http.HandlerFunc(handler.Ready))

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	wrapped.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(recorder.Body.String(), pingErr.Error()) {
		t.Fatal("ping error leaked into response")
	}

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	loggedError, _ := record["error"].(string)
	if !strings.Contains(loggedError, "ping database") || !strings.Contains(loggedError, pingErr.Error()) {
		t.Fatalf("logged error = %q, want wrapped ping error", loggedError)
	}
}
