package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestIsHTMLContent tests HTML content type detection
func TestIsHTMLContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "text/html",
			contentType: "text/html",
			want:        true,
		},
		{
			name:        "text/html with charset",
			contentType: "text/html; charset=utf-8",
			want:        true,
		},
		{
			name:        "application/xhtml+xml",
			contentType: "application/xhtml+xml",
			want:        true,
		},
		{
			name:        "application/xhtml",
			contentType: "application/xhtml",
			want:        true,
		},
		{
			name:        "uppercase TEXT/HTML",
			contentType: "TEXT/HTML",
			want:        true,
		},
		{
			name:        "mixed case Text/Html",
			contentType: "Text/Html; charset=UTF-8",
			want:        true,
		},
		{
			name:        "empty string",
			contentType: "",
			want:        false,
		},
		{
			name:        "application/json",
			contentType: "application/json",
			want:        false,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "image/png",
			contentType: "image/png",
			want:        false,
		},
		{
			name:        "with whitespace",
			contentType: "  text/html  ",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHTMLContent(tt.contentType)
			if got != tt.want {
				t.Errorf("isHTMLContent(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

// TestInjectPixel tests pixel injection into HTML
func assertPixelInjected(t *testing.T, result string, beforeTag string) {
	t.Helper()
	if !strings.Contains(result, `<img src="/px.gif`) {
		t.Errorf("should inject pixel, got: %s", result)
	}
	if beforeTag != "" {
		tagIndex := strings.Index(result, beforeTag)
		pixelIndex := strings.Index(result, `<img src="/px.gif`)
		if pixelIndex >= tagIndex {
			t.Errorf("pixel should be injected before %s tag", beforeTag)
		}
	}
}

func TestInjectPixel(t *testing.T) {
	t.Run("injects before closing body tag", func(t *testing.T) {
		html := []byte("<html><body><h1>Hello</h1></body></html>")
		req := httptest.NewRequest(http.MethodGet, "/test?utm_source=test", nil)
		result := string(injectPixel(html, req, nil))
		assertPixelInjected(t, result, "</body>")
		if !strings.Contains(result, `<img src="/px.gif?e=pageview&amp;auto=1&amp;url=`) {
			t.Errorf("should inject pixel with proper URL encoding, got: %s", result)
		}
		if !strings.Contains(result, `width="1" height="1" style="display:none"`) {
			t.Error("pixel should have proper attributes")
		}
	})

	t.Run("injects before closing html tag when no body tag", func(t *testing.T) {
		html := []byte("<html><div>Content</div></html>")
		req := httptest.NewRequest(http.MethodGet, "/page", nil)
		result := string(injectPixel(html, req, nil))
		assertPixelInjected(t, result, "</html>")
	})

	t.Run("appends to end when no closing tags", func(t *testing.T) {
		html := []byte("<div>Content without closing tags")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		result := string(injectPixel(html, req, nil))
		assertPixelInjected(t, result, "")
		if !strings.HasSuffix(strings.TrimSpace(result), `alt="">`) {
			t.Error("pixel should be appended to end")
		}
	})

	t.Run("includes HMAC script when auth configured", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		auth := NewHMACAuth("test-secret", "")
		result := string(injectPixel(html, req, auth))
		if !strings.Contains(result, `<script src="/hmac.js"></script>`) {
			t.Error("should include HMAC script")
		}
		assertPixelInjected(t, result, "")
	})

	t.Run("handles case insensitive closing tags", func(t *testing.T) {
		html := []byte("<html><body>Test</BODY></html>")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		result := string(injectPixel(html, req, nil))
		assertPixelInjected(t, result, "")
		if !strings.Contains(result, "</body>") && !strings.Contains(result, "</BODY>") {
			t.Error("should preserve body closing tag (case may change)")
		}
	})

	t.Run("escapes special characters in URL", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/test?q=foo&bar=baz<script>", nil)
		result := string(injectPixel(html, req, nil))
		if strings.Contains(result, "<script>") && !strings.Contains(result, `%3Cscript%3E`) {
			t.Error("special characters should be escaped in URL")
		}
	})

	t.Run("handles path without query string", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/simple", nil)
		result := string(injectPixel(html, req, nil))
		if !strings.Contains(result, `url=%2Fsimple"`) {
			t.Error("should encode simple path")
		}
	})
}

// TestIsTrackingPath tests tracking path detection
func TestIsTrackingPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/px.gif", true},
		{"/collect", true},
		{"/healthz", true},
		{"/readyz", true},
		{"/metrics", true},
		{"/hmac.js", true},
		{"/hmac/public-key", true},
		{"/pixel.js", true},
		{"/pixel.umd.js", true},
		{"/pixel.esm.js", true},
		{"/", false},
		{"/index.html", false},
		{"/api/users", false},
		{"/px.gif/extra", false},
		{"/collect/extra", false},
		{"/health", false},
		{"/hmac", false},
		{"/pixel", false},
		{"/pixel.min.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTrackingPath(tt.path)
			if got != tt.want {
				t.Errorf("isTrackingPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestNewProxyHandler tests proxy handler creation
func TestNewProxyHandler(t *testing.T) {
	t.Run("creates handler with destination", func(t *testing.T) {
		handler := NewProxyHandler("http://example.com", nil)

		if handler == nil {
			t.Fatal("handler should not be nil")
		}

		if handler.destination != "http://example.com" {
			t.Errorf("destination = %q, want http://example.com", handler.destination)
		}

		if handler.hmacAuth != nil {
			t.Error("hmacAuth should be nil")
		}

		if handler.client == nil {
			t.Error("client should not be nil")
		}

		if handler.client.Timeout != 30*time.Second {
			t.Errorf("client timeout = %v, want 30s", handler.client.Timeout)
		}
	})

	t.Run("creates handler with auto inject and HMAC", func(t *testing.T) {
		auth := NewHMACAuth("secret", "")
		handler := NewProxyHandler("http://example.com", auth)

		if handler.hmacAuth == nil {
			t.Error("hmacAuth should not be nil")
		}
	})
}

// TestProxyHandlerServeHTTP tests proxy request forwarding
func TestProxyHandlerServeHTTP(t *testing.T) {
	t.Run("proxies request to destination", func(t *testing.T) {
		// Create a test backend server
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Header", "test-value")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("backend response"))
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		if w.Header().Get("X-Test-Header") != "test-value" {
			t.Error("should copy response headers from backend")
		}

		body := w.Body.String()
		if body != "backend response" {
			t.Errorf("body = %q, want 'backend response'", body)
		}
	})

	t.Run("copies request headers", func(t *testing.T) {
		var receivedHeaders http.Header
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Custom-Header", "custom-value")
		req.Header.Set("User-Agent", "TestAgent/1.0")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
			t.Error("should forward custom headers")
		}

		if receivedHeaders.Get("User-Agent") != "TestAgent/1.0" {
			t.Error("should forward User-Agent header")
		}
	})

	t.Run("forwards query parameters", func(t *testing.T) {
		var receivedQuery string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		req := httptest.NewRequest(http.MethodGet, "/test?param1=value1&param2=value2", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if receivedQuery != "param1=value1&param2=value2" {
			t.Errorf("query = %q, want 'param1=value1&param2=value2'", receivedQuery)
		}
	})

	t.Run("injects pixel into HTML response", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><body>Test content</body></html>"))
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		req := httptest.NewRequest(http.MethodGet, "/page", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `<img src="/px.gif`) {
			t.Error("should inject pixel into HTML response")
		}

		if !strings.Contains(body, "Test content") {
			t.Error("should preserve original content")
		}
	})

	t.Run("does not inject pixel into non-HTML response", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if strings.Contains(body, `<img src="/px.gif`) {
			t.Error("should not inject pixel into JSON response")
		}

		if body != `{"status":"ok"}` {
			t.Errorf("body = %q, want original JSON", body)
		}
	})

	t.Run("handles invalid destination URL", func(t *testing.T) {
		handler := NewProxyHandler("://invalid-url", nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handles unreachable backend", func(t *testing.T) {
		// Use an invalid port
		handler := NewProxyHandler("http://localhost:0", nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadGateway {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadGateway)
		}
	})

	t.Run("forwards POST request with body", func(t *testing.T) {
		var receivedBody []byte
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		handler := NewProxyHandler(backend.URL, nil)

		body := bytes.NewReader([]byte("test body data"))
		req := httptest.NewRequest(http.MethodPost, "/api", body)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if string(receivedBody) != "test body data" {
			t.Errorf("received body = %q, want 'test body data'", receivedBody)
		}
	})
}

// TestNewMiddlewareRouter tests middleware router creation
func TestNewMiddlewareRouter(t *testing.T) {
	mux := http.NewServeMux()
	router := NewMiddlewareRouter(mux, "http://example.com", nil, nil)

	if router == nil {
		t.Fatal("router should not be nil")
	}

	if router.trackingMux != mux {
		t.Error("trackingMux should be set")
	}

	if router.proxy == nil {
		t.Error("proxy should not be nil")
	}
}

// TestMiddlewareRouterServeHTTP tests routing behavior
func TestMiddlewareRouterServeHTTP(t *testing.T) {
	t.Run("routes tracking paths to tracking mux", func(t *testing.T) {
		trackingCalled := false
		mux := http.NewServeMux()
		mux.HandleFunc("/px.gif", func(w http.ResponseWriter, r *http.Request) {
			trackingCalled = true
			w.WriteHeader(http.StatusOK)
		})

		// Create a backend that should NOT be called
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("backend should not be called for tracking paths")
		}))
		defer backend.Close()

		router := NewMiddlewareRouter(mux, backend.URL, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if !trackingCalled {
			t.Error("tracking handler should have been called")
		}
	})

	t.Run("proxies non-tracking paths to backend", func(t *testing.T) {
		backendCalled := false
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backendCalled = true
			w.Write([]byte("backend response"))
		}))
		defer backend.Close()

		mux := http.NewServeMux()
		router := NewMiddlewareRouter(mux, backend.URL, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/app/page", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if !backendCalled {
			t.Error("backend should have been called")
		}

		body := w.Body.String()
		if body != "backend response" {
			t.Errorf("body = %q, want 'backend response'", body)
		}
	})

	t.Run("routes all standard tracking paths", func(t *testing.T) {
		trackingPaths := []string{"/px.gif", "/collect", "/healthz", "/readyz", "/metrics", "/hmac.js", "/hmac/public-key", "/pixel.js", "/pixel.umd.js", "/pixel.esm.js"}

		for _, path := range trackingPaths {
			t.Run(path, func(t *testing.T) {
				called := false
				mux := http.NewServeMux()
				mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusOK)
				})

				backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("backend should not be called for tracking path %s", path)
				}))
				defer backend.Close()

				router := NewMiddlewareRouter(mux, backend.URL, nil, nil)

				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				if !called {
					t.Errorf("tracking handler should have been called for %s", path)
				}
			})
		}
	})
}
