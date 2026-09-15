package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObservabilityMiddlewareRecordsRequestMetrics(t *testing.T) {
	metrics := newRequestMetrics()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := observabilityMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}), logger, metrics)

	request := httptest.NewRequest(http.MethodGet, "/students", nil)
	request.Header.Set("x-correlation-id", "corr-123")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if metrics.totalRequests.Load() != 1 {
		t.Fatalf("total requests = %d, want 1", metrics.totalRequests.Load())
	}
	if metrics.serverErrors.Load() != 1 {
		t.Fatalf("server errors = %d, want 1", metrics.serverErrors.Load())
	}
	if metrics.inFlight.Load() != 0 {
		t.Fatalf("in-flight requests = %d, want 0", metrics.inFlight.Load())
	}
}

func TestMetricsHandlerExposesPrometheusValues(t *testing.T) {
	metrics := newRequestMetrics()
	metrics.totalRequests.Store(4)
	metrics.serverErrors.Store(1)
	metrics.inFlight.Store(2)
	recorder := httptest.NewRecorder()

	metrics.handler(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	for _, expected := range []string{
		"http_requests_total 4",
		"http_requests_5xx_total 1",
		"http_requests_in_flight 2",
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("metrics response does not contain %q", expected)
		}
	}
}
