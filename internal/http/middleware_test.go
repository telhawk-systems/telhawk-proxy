package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shortontech/gotrack/internal/metrics"
)

// TestRequestLogger tests the request logging middleware
func TestRequestLogger(t *testing.T) {
	t.Run("calls next handler", func(t *testing.T) {
		handlerCalled := false
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := RequestLogger(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("next handler should have been called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("logs request details", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := RequestLogger(nextHandler)

		req := httptest.NewRequest(http.MethodPost, "/api/test?query=value", nil)
		req.Header.Set("User-Agent", "TestAgent/1.0")
		w := httptest.NewRecorder()

		// Note: Actual log output goes to default logger, which we can't easily capture
		// but we can verify the middleware executes without panicking
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("handles errors from next handler", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		middleware := RequestLogger(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("measures request duration", func(t *testing.T) {
		// Create a handler that takes some time
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		middleware := RequestLogger(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		middleware.ServeHTTP(w, req)
		elapsed := time.Since(start)

		// Verify it took at least as long as the handler sleep
		if elapsed < 10*time.Millisecond {
			t.Errorf("elapsed time = %v, should be at least 10ms", elapsed)
		}
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				middleware := RequestLogger(nextHandler)

				req := httptest.NewRequest(method, "/test", nil)
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
				}
			})
		}
	})
}

// TestCors tests the CORS middleware
func TestCors(t *testing.T) {
	t.Run("sets CORS headers for GET request", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := cors(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		headers := w.Header()

		if got := headers.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}

		if got := headers.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
			t.Errorf("Access-Control-Allow-Headers should contain Content-Type, got %q", got)
		}

		if got := headers.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
			t.Errorf("Access-Control-Allow-Methods should contain GET, got %q", got)
		}
	})

	t.Run("handles OPTIONS preflight request", func(t *testing.T) {
		handlerCalled := false
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := cors(nextHandler)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("next handler should not be called for OPTIONS requests")
		}

		if w.Code != http.StatusNoContent {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNoContent)
		}

		// Verify CORS headers are still set
		headers := w.Header()
		if got := headers.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}
	})

	t.Run("calls next handler for POST request", func(t *testing.T) {
		handlerCalled := false
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusCreated)
		})

		middleware := cors(nextHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("next handler should be called for POST requests")
		}

		if w.Code != http.StatusCreated {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("sets correct allow methods header", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := cors(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		allowMethods := w.Header().Get("Access-Control-Allow-Methods")
		expectedMethods := []string{"GET", "POST", "OPTIONS"}

		for _, method := range expectedMethods {
			if !strings.Contains(allowMethods, method) {
				t.Errorf("Access-Control-Allow-Methods should contain %s, got %q", method, allowMethods)
			}
		}
	})

	t.Run("sets correct allow headers", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := cors(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		allowHeaders := w.Header().Get("Access-Control-Allow-Headers")

		if !strings.Contains(allowHeaders, "Content-Type") {
			t.Errorf("should contain Content-Type, got %q", allowHeaders)
		}
		if !strings.Contains(allowHeaders, "DNT") {
			t.Errorf("should contain DNT, got %q", allowHeaders)
		}
	})
}

// TestResponseWriter tests the responseWriter wrapper
func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		rw.WriteHeader(http.StatusCreated)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusCreated)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("underlying recorder Code = %d, want %d", recorder.Code, http.StatusCreated)
		}
	})

	t.Run("defaults to 200 OK", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		// Write without calling WriteHeader
		rw.Write([]byte("test"))

		// Should still have default status
		if rw.statusCode != http.StatusOK {
			t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusOK)
		}
	})

	t.Run("captures various status codes", func(t *testing.T) {
		testCases := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusNoContent,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				recorder := httptest.NewRecorder()
				rw := &responseWriter{
					ResponseWriter: recorder,
					statusCode:     http.StatusOK,
				}

				rw.WriteHeader(statusCode)

				if rw.statusCode != statusCode {
					t.Errorf("statusCode = %d, want %d", rw.statusCode, statusCode)
				}
			})
		}
	})

	t.Run("embeds ResponseWriter correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		// Should be able to use embedded methods
		rw.Header().Set("X-Test", "value")

		if got := recorder.Header().Get("X-Test"); got != "value" {
			t.Errorf("header X-Test = %q, want value", got)
		}
	})
}

// TestMetricsMiddleware tests the metrics tracking middleware
func TestMetricsMiddleware(t *testing.T) {
	// Use InitMetrics which returns existing instance or creates new one
	m := metrics.InitMetrics()

	t.Run("handles nil metrics gracefully", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := MetricsMiddleware(nil)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Should not panic
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("records metrics for successful request", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := MetricsMiddleware(m)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Verify the handler was called
		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		// Note: We can't easily verify Prometheus metrics without using testutil
		// but we can confirm no panics occurred
	})

	t.Run("records metrics for error request", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		middleware := MetricsMiddleware(m)(nextHandler)

		req := httptest.NewRequest(http.MethodPost, "/api/error", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("captures status code from handler", func(t *testing.T) {
		testCases := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, statusCode := range testCases {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				})

				middleware := MetricsMiddleware(m)(nextHandler)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)

				if w.Code != statusCode {
					t.Errorf("status code = %d, want %d", w.Code, statusCode)
				}
			})
		}
	})

	t.Run("measures request duration", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		middleware := MetricsMiddleware(m)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		middleware.ServeHTTP(w, req)
		elapsed := time.Since(start)

		// Verify it took at least as long as the handler sleep
		if elapsed < 10*time.Millisecond {
			t.Errorf("elapsed time = %v, should be at least 10ms", elapsed)
		}
	})

	t.Run("tracks different endpoints", func(t *testing.T) {
		endpoints := []string{"/", "/api/test", "/collect", "/healthz"}

		for _, endpoint := range endpoints {
			t.Run(endpoint, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				middleware := MetricsMiddleware(m)(nextHandler)

				req := httptest.NewRequest(http.MethodGet, endpoint, nil)
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
				}
			})
		}
	})

	t.Run("tracks different methods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				middleware := MetricsMiddleware(m)(nextHandler)

				req := httptest.NewRequest(method, "/test", nil)
				w := httptest.NewRecorder()

				middleware.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
				}
			})
		}
	})

	t.Run("does not modify response", func(t *testing.T) {
		expectedBody := "test response body"
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Header", "custom-value")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedBody))
		})

		middleware := MetricsMiddleware(m)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		if got := w.Header().Get("X-Custom-Header"); got != "custom-value" {
			t.Errorf("X-Custom-Header = %q, want custom-value", got)
		}

		if got := w.Body.String(); got != expectedBody {
			t.Errorf("body = %q, want %q", got, expectedBody)
		}
	})
}

// TestMiddlewareChaining tests that middleware can be chained together
func TestMiddlewareChaining(t *testing.T) {
	// Use InitMetrics to avoid registry conflicts
	m := metrics.InitMetrics()

	t.Run("chains RequestLogger and cors", func(t *testing.T) {
		handlerCalled := false
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		// Chain: RequestLogger -> cors -> finalHandler
		handler := RequestLogger(cors(finalHandler))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("final handler should have been called")
		}

		// Verify CORS headers are set
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}
	})

	t.Run("chains all three middleware", func(t *testing.T) {
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Chain: RequestLogger -> MetricsMiddleware -> cors -> finalHandler
		handler := RequestLogger(MetricsMiddleware(m)(cors(finalHandler)))

		req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader([]byte("test")))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		// Verify CORS headers
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
		}
	})
}
