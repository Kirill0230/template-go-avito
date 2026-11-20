package push_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kirill0230/template-go-avito/internal/gateway/push"
)

func TestGatewayRequestPosition(t *testing.T) {
	driverID, tripID := uuid.New(), uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/push" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			RequestID string            `json:"request_id"`
			Recipient uuid.UUID         `json:"recipient_id"`
			Kind      string            `json:"kind"`
			Data      map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, err := uuid.Parse(body.RequestID); err != nil {
			t.Errorf("invalid request_id: %v", err)
		}
		if body.Recipient != driverID || body.Kind != "REQUEST_POSITION" || body.Data["trip_id"] != tripID.String() {
			t.Errorf("unexpected body: %#v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	gateway := push.NewGateway(server.URL, time.Second)
	defer gateway.Close()
	if err := gateway.RequestPosition(context.Background(), driverID, tripID); err != nil {
		t.Fatalf("RequestPosition() error: %v", err)
	}
}

func TestGatewayDistinguishesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	gateway := push.NewGateway(server.URL, time.Second)
	defer gateway.Close()

	err := gateway.RequestPosition(context.Background(), uuid.New(), uuid.New())
	var statusErr *push.StatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error = %v, want StatusError 503", err)
	}
}
