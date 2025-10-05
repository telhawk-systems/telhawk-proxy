package httpx

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"revinar.io/go.track/internal/metrics"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s ua=%q dur=%s", r.Method, r.URL.Path, r.UserAgent(), time.Since(start))
	})
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Very permissive for dev; tighten in production.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, DNT")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware adds HTTP request metrics tracking
func MetricsMiddleware(appMetrics *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if appMetrics == nil {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			endpoint := r.URL.Path
			method := r.Method
			status := strconv.Itoa(wrapped.statusCode)

			// Record metrics
			appMetrics.IncrementHTTPRequests(endpoint, method, status)
			appMetrics.ObserveHTTPDuration(endpoint, method, duration)
		})
	}
}
