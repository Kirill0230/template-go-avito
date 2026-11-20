package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("push service returned HTTP %d", e.StatusCode)
}

type Gateway struct {
	baseURL   string
	client    *http.Client
	transport *http.Transport
}

func NewGateway(baseURL string, timeout time.Duration) *Gateway {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
	}

	return &Gateway{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(transport),
		},
		transport: transport,
	}
}

func (g *Gateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	payload := struct {
		RequestID   string            `json:"request_id"`
		RecipientID uuid.UUID         `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID.String()},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode push request: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.baseURL+"/api/v1/push",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("send push request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode != http.StatusAccepted {
		return &StatusError{StatusCode: response.StatusCode}
	}

	return nil
}

func (g *Gateway) Close() {
	g.transport.CloseIdleConnections()
}
