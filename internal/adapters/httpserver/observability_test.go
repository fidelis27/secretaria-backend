package httpserver

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
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

func TestMetricsHandlerRequiresSuperAdmin(t *testing.T) {
	metrics := newRequestMetrics()
	handler := metricsHandler(metrics)

	for _, test := range []struct {
		name       string
		user       domainuser.User
		wantStatus int
	}{
		{name: "regular user", user: domainuser.User{ID: "user-1", Status: "active"}, wantStatus: http.StatusForbidden},
		{name: "superadmin", user: domainuser.User{ID: "user-2", Status: "active", SuperAdmin: true}, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, test.user))

			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestSecurityHeadersMiddlewareAddsBrowserHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing X-Content-Type-Options nosniff")
	}
	if recorder.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("missing Referrer-Policy")
	}
}

func TestObservabilityLogsDoNotContainAuthorizationOrFullEmail(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	handler := observabilityMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}), logger, newRequestMetrics())
	request := httptest.NewRequest(http.MethodGet, "/users?q=private@example.com", nil)
	request.Header.Set("Authorization", "Bearer "+strings.Join([]string{"test", "token"}, "-"))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	for _, secret := range []string{strings.Join([]string{"test", "token"}, "-"), "private@example.com", "Authorization"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("request log contains sensitive value %q: %s", secret, output.String())
		}
	}
}
