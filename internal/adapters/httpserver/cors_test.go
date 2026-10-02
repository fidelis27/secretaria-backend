package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsLocalFrontendOriginUsesConfiguredOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ORIGINS", "https://host.example.com, http://localhost:9000")

	if !isLocalFrontendOrigin("https://host.example.com") {
		t.Fatal("configured origin should be allowed")
	}
	if !isLocalFrontendOrigin("http://localhost:9000") {
		t.Fatal("trimmed configured origin should be allowed")
	}
	if isLocalFrontendOrigin("http://localhost:4174") {
		t.Fatal("unconfigured origin should be denied")
	}
}

func TestIsLocalFrontendOriginUsesLocalDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ORIGINS", "")

	if !isLocalFrontendOrigin("http://localhost:4174") {
		t.Fatal("localhost should be allowed outside production")
	}
	if isLocalFrontendOrigin("https://untrusted.example.com") {
		t.Fatal("untrusted origin should be denied by default")
	}
}

func TestIsLocalFrontendOriginRejectsLocalhostInProductionWithoutExplicitOrigin(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ORIGINS", "")

	if isLocalFrontendOrigin("http://localhost:4174") {
		t.Fatal("localhost should be denied in production when not explicitly configured")
	}
}

func TestCorsMiddlewareHandlesAllowedPreflight(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	handler := corsMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		t.Fatal("preflight should not reach the wrapped handler")
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/students", nil)
	request.Header.Set("Origin", "http://localhost:4174")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4174" {
		t.Fatalf("allow-origin = %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
	if recorder.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, DELETE, OPTIONS" {
		t.Fatalf("allow-methods = %q", recorder.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ana","extra":true}`))

	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(recorder, request, &input); err == nil {
		t.Fatal("decodeJSON should reject unknown fields")
	}
}

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	server := newHTTPServer(":3333", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout = %s", server.ReadTimeout)
	}
	if server.WriteTimeout != 30*time.Second {
		t.Fatalf("WriteTimeout = %s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s", server.IdleTimeout)
	}
}
