package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Kirill0230/template-go-avito/internal/http/middleware"
)

func TestHTTPDurationRecordedOnceForEveryResponse(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousProvider)
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("shutdown meter provider: %v", err)
		}
	})

	httpMetrics, err := middleware.NewHTTPMetrics(provider.Meter("test"))
	if err != nil {
		t.Fatalf("create HTTP metrics: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router := chi.NewRouter()
	router.Use(
		middleware.Tracing,
		middleware.Metrics(httpMetrics),
		middleware.Recover(logger),
	)
	router.Get("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.Get("/bad", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodGet, path: "/ok", status: http.StatusOK},
		{method: http.MethodGet, path: "/bad", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{method: http.MethodPost, path: "/ok", status: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tt.method, tt.path, nil)
		router.ServeHTTP(response, request)
		if response.Code != tt.status {
			t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.status)
		}
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	histogram := findHTTPDurationHistogram(t, collected)

	counts := make(map[int64]uint64)
	var total uint64
	for _, point := range histogram.DataPoints {
		status, ok := point.Attributes.Value("http.response.status_code")
		if !ok {
			t.Fatal("duration data point has no status code")
		}
		counts[status.AsInt64()] += point.Count
		total += point.Count
	}

	if total != uint64(len(tests)) {
		t.Fatalf("duration observations = %d, want %d", total, len(tests))
	}
	for _, tt := range tests {
		if counts[int64(tt.status)] != 1 {
			t.Fatalf("status %d observations = %d, want 1", tt.status, counts[int64(tt.status)])
		}
	}
}

func findHTTPDurationHistogram(
	t *testing.T,
	metrics metricdata.ResourceMetrics,
) metricdata.Histogram[float64] {
	t.Helper()

	wantBounds := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "http.server.request.duration" {
				continue
			}
			histogram, ok := candidate.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("duration metric has unexpected data: %#v", candidate.Data)
			}
			for _, point := range histogram.DataPoints {
				if !slices.Equal(point.Bounds, wantBounds) {
					t.Fatalf("duration bounds = %v, want %v", point.Bounds, wantBounds)
				}
			}
			return histogram
		}
	}

	t.Fatal("http.server.request.duration metric not found")
	return metricdata.Histogram[float64]{}
}
