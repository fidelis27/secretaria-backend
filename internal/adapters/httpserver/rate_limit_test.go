package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

func TestRateLimitMiddlewareReturnsRetryAfter(t *testing.T) {
	limiter := newRequestRateLimiter(1)
	handler := rateLimitMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}), limiter)

	first := httptest.NewRequest(http.MethodPost, "/students", nil)
	first.RemoteAddr = "192.0.2.1:1234"
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, first)
	if firstRecorder.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusNoContent)
	}

	second := httptest.NewRequest(http.MethodPost, "/students", nil)
	second.RemoteAddr = "192.0.2.1:4321"
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, second)
	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusTooManyRequests)
	}
	if secondRecorder.Header().Get("Retry-After") == "" {
		t.Fatal("429 response did not include Retry-After")
	}
}

func TestRateLimiterIsSafeForConcurrentRequests(t *testing.T) {
	limiter := newRequestRateLimiter(5)
	var allowed atomic.Int32
	var workers sync.WaitGroup
	for range 50 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ok, _ := limiter.allow("user:user-1", time.Now())
			if ok {
				allowed.Add(1)
			}
		}()
	}
	workers.Wait()
	if got := allowed.Load(); got != 5 {
		t.Fatalf("allowed %d concurrent requests, want 5", got)
	}
}

func TestRateLimitKeyPrefersAuthenticatedUser(t *testing.T) {
	user := domainuser.User{ID: "user-123"}
	request := httptest.NewRequest(http.MethodPost, "/students", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	request = request.WithContext(context.WithValue(request.Context(), identityContextKey{}, user))
	if got := rateLimitKey(request); got != "user:user-123" {
		t.Fatalf("rate limit key = %q, want user identity", got)
	}
}
