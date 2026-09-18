package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLocalFrontendOriginUsesConfiguredOrigins(t *testing.T) {
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
	t.Setenv("CORS_ORIGINS", "")

	if !isLocalFrontendOrigin("http://localhost:4174") {
		t.Fatal("Host local origin should be allowed by default")
	}
	if isLocalFrontendOrigin("https://untrusted.example.com") {
		t.Fatal("untrusted origin should be denied by default")
	}
}

func TestIsLocalFrontendOriginAllowsVercelDomains(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")

	if !isLocalFrontendOrigin("https://mfe-communication-mfe-student.vercel.app") {
		t.Fatal("project Vercel domain should be allowed")
	}
	if !isLocalFrontendOrigin("https://mfe-communication-mfe-student-git-main-fidelis27s-projects.vercel.app") {
		t.Fatal("Vercel preview domain should be allowed")
	}
	if isLocalFrontendOrigin("https://malicious.example.com") {
		t.Fatal("non-Vercel origin should be denied")
	}
}

func TestCorsMiddlewareHandlesAllowedPreflight(t *testing.T) {
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
}
