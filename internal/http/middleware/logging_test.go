package middleware_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace"

	api "github.com/Kirill0230/template-go-avito/internal/generated"
	"github.com/Kirill0230/template-go-avito/internal/http/middleware"
	"github.com/Kirill0230/template-go-avito/internal/http/response"
)

func TestLoggingIncludesInternalErrorWithoutLeakingIt(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	internalErr := errors.New("database password leaked")

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problem := api.Problem{
			Title:  http.StatusText(http.StatusInternalServerError),
			Status: http.StatusInternalServerError,
			Code:   "internal_error",
		}
		response.WriteProblem(w, r, http.StatusInternalServerError, problem, internalErr)
	})
	handler := middleware.Logging(logger)(middleware.Recover(logger)(endpoint))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if strings.Contains(recorder.Body.String(), internalErr.Error()) {
		t.Fatal("internal error leaked into response")
	}

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	if record["level"] != "ERROR" {
		t.Fatalf("log level = %v, want ERROR", record["level"])
	}
	if record["error"] != internalErr.Error() {
		t.Fatalf("logged error = %v, want %q", record["error"], internalErr)
	}
}

func TestPanicIsLoggedAndMeasured(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("shutdown meter provider: %v", err)
		}
	})

	httpMetrics, err := middleware.NewHTTPMetrics(provider.Meter("test"))
	if err != nil {
		t.Fatalf("create HTTP metrics: %v", err)
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	endpoint := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("unexpected failure")
	})
	handler := middleware.Logging(logger)(
		middleware.Metrics(httpMetrics)(
			middleware.Recover(logger)(endpoint),
		),
	)

	traceID := trace.TraceID{1}
	spanID := trace.SpanID{2}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	requestContext := trace.ContextWithSpanContext(ctx, spanContext)
	request := httptest.NewRequest(http.MethodGet, "/panic", nil).WithContext(requestContext)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	records := make(map[string]map[string]any)
	scanner := bufio.NewScanner(&logs)
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("decode log record: %v", err)
		}
		message, _ := record["msg"].(string)
		records[message] = record
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan logs: %v", err)
	}

	for _, message := range []string{"panic recovered", "request handled"} {
		record := records[message]
		if record == nil {
			t.Fatalf("log %q not found", message)
		}
		if record["trace_id"] != traceID.String() || record["span_id"] != spanID.String() {
			t.Fatalf("log %q has trace_id=%v span_id=%v", message, record["trace_id"], record["span_id"])
		}
	}
	if records["request handled"]["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("logged status = %v, want 500", records["request handled"]["status"])
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	if !hasDurationMetricWithStatus(collected, http.StatusInternalServerError) {
		t.Fatal("request duration metric with status 500 not found")
	}
}

func hasDurationMetricWithStatus(metrics metricdata.ResourceMetrics, status int64) bool {
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "http.server.request.duration" {
				continue
			}

			histogram, ok := candidate.Data.(metricdata.Histogram[float64])
			if !ok {
				continue
			}
			for _, point := range histogram.DataPoints {
				value, ok := point.Attributes.Value(attribute.Key("http.response.status_code"))
				if ok && value.AsInt64() == status && point.Count == 1 {
					return true
				}
			}
		}
	}

	return false
}
