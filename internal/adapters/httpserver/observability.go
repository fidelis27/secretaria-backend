package httpserver

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type requestMetrics struct {
	totalRequests atomic.Uint64
	serverErrors  atomic.Uint64
	inFlight      atomic.Int64
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{}
}

func (metrics *requestMetrics) handler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(writer, "http_requests_total %d\n", metrics.totalRequests.Load())
	fmt.Fprintf(writer, "http_requests_5xx_total %d\n", metrics.serverErrors.Load())
	fmt.Fprintf(writer, "http_requests_in_flight %d\n", metrics.inFlight.Load())
}

func observabilityMiddleware(next http.Handler, logger *slog.Logger, metrics *requestMetrics) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		metrics.inFlight.Add(1)
		defer metrics.inFlight.Add(-1)

		recorder := &statusRecorder{ResponseWriter: writer}
		next.ServeHTTP(recorder, request)

		status := recorder.status
		metrics.totalRequests.Add(1)
		if status >= http.StatusInternalServerError {
			metrics.serverErrors.Add(1)
		}
		logger.Info("http_request",
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("correlation_id", request.Header.Get("x-correlation-id")),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := recorder.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (recorder *statusRecorder) Flush() {
	flusher, ok := recorder.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.ResponseWriter.Write(body)
}
