package httpserver

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateBucket struct {
	tokens  float64
	last    time.Time
	touched time.Time
}

type requestRateLimiter struct {
	mutex             sync.Mutex
	requestsPerMinute int
	buckets           map[string]rateBucket
	calls             uint64
}

func newRequestRateLimiter(requestsPerMinute int) *requestRateLimiter {
	return &requestRateLimiter{
		requestsPerMinute: requestsPerMinute,
		buckets:           make(map[string]rateBucket),
	}
}

func (limiter *requestRateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	rate := float64(limiter.requestsPerMinute) / 60
	bucket, found := limiter.buckets[key]
	if !found {
		bucket = rateBucket{tokens: float64(limiter.requestsPerMinute), last: now}
	}
	bucket.tokens = math.Min(float64(limiter.requestsPerMinute), bucket.tokens+now.Sub(bucket.last).Seconds()*rate)
	bucket.last = now
	bucket.touched = now
	limiter.calls++
	if limiter.calls%256 == 0 {
		for bucketKey, oldBucket := range limiter.buckets {
			if now.Sub(oldBucket.touched) > 10*time.Minute {
				delete(limiter.buckets, bucketKey)
			}
		}
	}

	if bucket.tokens >= 1 {
		bucket.tokens--
		limiter.buckets[key] = bucket
		return true, 0
	}

	retryAfter := time.Duration((1 - bucket.tokens) / rate * float64(time.Second))
	limiter.buckets[key] = bucket
	return false, retryAfter
}

func rateLimitMiddleware(next http.Handler, limiter *requestRateLimiter) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !isRateLimitedRequest(request) {
			next.ServeHTTP(writer, request)
			return
		}

		key := rateLimitKey(request)
		allowed, retryAfter := limiter.allow(key, time.Now())
		if !allowed {
			retrySeconds := int(math.Ceil(retryAfter.Seconds()))
			if retrySeconds < 1 {
				retrySeconds = 1
			}
			writer.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			writeError(writer, http.StatusTooManyRequests, "limite de requisicoes excedido")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func isRateLimitedRequest(request *http.Request) bool {
	switch request.Method {
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return request.URL.Path == "/events"
	}
}

func rateLimitKey(request *http.Request) string {
	if user, ok := userFromContext(request.Context()); ok {
		return "user:" + user.ID
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return "ip:" + host
	}
	return "ip:" + request.RemoteAddr
}
