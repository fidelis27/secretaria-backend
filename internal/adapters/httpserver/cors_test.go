package httpserver

import "testing"

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
