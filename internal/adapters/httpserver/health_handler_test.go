package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apphealth "github.com/fidelis27/secretaria-backend/internal/application/health"
)

type healthCheckerStub struct {
	err error
}

func (stub healthCheckerStub) Check(context.Context) error {
	return stub.err
}

func TestHealthHandlerReturns503WhenUnhealthy(t *testing.T) {
	handler := healthHandler(apphealth.NewService(healthCheckerStub{err: errors.New("ping MariaDB: connection refused")}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
