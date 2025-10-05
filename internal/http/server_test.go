package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shortontech/gotrack/internal/event"
	"github.com/shortontech/gotrack/internal/metrics"
	"github.com/shortontech/gotrack/pkg/config"
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
func TestInjectPixel(t *testing.T) {
	t.Run("injects before closing body tag", func(t *testing.T) {
		html := []byte("<html><body><h1>Hello</h1></body></html>")
		req := httptest.NewRequest(http.MethodGet, "/test?utm_source=test", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		// Check that pixel is injected with proper URL encoding
		// Note: HTML escaping converts & to &amp;
		if !strings.Contains(resultStr, `<img src="/px.gif?e=pageview&amp;auto=1&amp;url=`) {
			t.Errorf("should inject pixel, got: %s", resultStr)
		}

		if !strings.Contains(resultStr, `width="1" height="1" style="display:none"`) {
			t.Error("pixel should have proper attributes")
		}

		// Check that pixel is before </body>
		bodyCloseIndex := strings.Index(resultStr, "</body>")
		pixelIndex := strings.Index(resultStr, `<img src="/px.gif`)
		if pixelIndex >= bodyCloseIndex {
			t.Error("pixel should be injected before </body> tag")
		}
	})

	t.Run("injects before closing html tag when no body tag", func(t *testing.T) {
		html := []byte("<html><div>Content</div></html>")
		req := httptest.NewRequest(http.MethodGet, "/page", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		if !strings.Contains(resultStr, `<img src="/px.gif`) {
			t.Error("should inject pixel")
		}

		// Check that pixel is before </html>
		htmlCloseIndex := strings.Index(resultStr, "</html>")
		pixelIndex := strings.Index(resultStr, `<img src="/px.gif`)
		if pixelIndex >= htmlCloseIndex {
			t.Error("pixel should be injected before </html> tag")
		}
	})

	t.Run("appends to end when no closing tags", func(t *testing.T) {
		html := []byte("<div>Content without closing tags")
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		if !strings.Contains(resultStr, `<img src="/px.gif`) {
			t.Error("should inject pixel")
		}

		if !strings.HasSuffix(strings.TrimSpace(resultStr), `alt="">`) {
			t.Error("pixel should be appended to end")
		}
	})

	t.Run("includes HMAC script when auth configured", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		auth := NewHMACAuth("test-secret", "", false)

		result := injectPixel(html, req, auth)
		resultStr := string(result)

		if !strings.Contains(resultStr, `<script src="/hmac.js"></script>`) {
			t.Error("should include HMAC script")
		}

		if !strings.Contains(resultStr, `<img src="/px.gif`) {
			t.Error("should still include pixel")
		}
	})

	t.Run("handles case insensitive closing tags", func(t *testing.T) {
		html := []byte("<html><body>Test</BODY></html>")
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		if !strings.Contains(resultStr, `<img src="/px.gif`) {
			t.Errorf("should inject pixel, got: %s", resultStr)
		}

		// The regex replaces </BODY> with pixel + </body>, so check for that
		if !strings.Contains(resultStr, "</body>") && !strings.Contains(resultStr, "</BODY>") {
			t.Error("should preserve body closing tag (case may change)")
		}

		// Pixel should be present before the closing tag
		bodyCloseIndex := strings.LastIndex(resultStr, "body>")
		pixelIndex := strings.Index(resultStr, `<img src="/px.gif`)
		if bodyCloseIndex < 0 {
			t.Error("body closing tag not found")
		} else if pixelIndex < 0 {
			t.Error("pixel not found")
		} else if pixelIndex >= bodyCloseIndex {
			t.Errorf("pixel should be injected before body closing tag, pixel at %d, body at %d", pixelIndex, bodyCloseIndex)
		}
	})

	t.Run("escapes special characters in URL", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/test?q=foo&bar=baz<script>", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		// URL should be properly escaped
		if strings.Contains(resultStr, "<script>") && !strings.Contains(resultStr, `%3Cscript%3E`) {
			t.Error("special characters should be escaped in URL")
		}
	})

	t.Run("handles path without query string", func(t *testing.T) {
		html := []byte("<html><body>Test</body></html>")
		req := httptest.NewRequest(http.MethodGet, "/simple", nil)

		result := injectPixel(html, req, nil)
		resultStr := string(result)

		if !strings.Contains(resultStr, `url=%2Fsimple"`) {
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
		{"/", false},
		{"/index.html", false},
		{"/api/users", false},
		{"/px.gif/extra", false},
		{"/collect/extra", false},
		{"/health", false},
		{"/hmac", false},
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
		handler := NewProxyHandler("http://example.com", false, nil)

		if handler == nil {
			t.Fatal("handler should not be nil")
		}

		if handler.destination != "http://example.com" {
			t.Errorf("destination = %q, want http://example.com", handler.destination)
		}

		if handler.autoInjectPixel {
			t.Error("autoInjectPixel should be false")
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
		auth := NewHMACAuth("secret", "", false)
		handler := NewProxyHandler("http://example.com", true, auth)

		if !handler.autoInjectPixel {
			t.Error("autoInjectPixel should be true")
		}

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

		handler := NewProxyHandler(backend.URL, false, nil)

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

		handler := NewProxyHandler(backend.URL, false, nil)

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

		handler := NewProxyHandler(backend.URL, false, nil)

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

		handler := NewProxyHandler(backend.URL, true, nil) // autoInjectPixel = true

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

		handler := NewProxyHandler(backend.URL, true, nil) // autoInjectPixel = true

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
		handler := NewProxyHandler("://invalid-url", false, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handles unreachable backend", func(t *testing.T) {
		// Use an invalid port
		handler := NewProxyHandler("http://localhost:0", false, nil)

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

		handler := NewProxyHandler(backend.URL, false, nil)

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
	router := NewMiddlewareRouter(mux, "http://example.com", false, nil)

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

		router := NewMiddlewareRouter(mux, backend.URL, false, nil)

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
		router := NewMiddlewareRouter(mux, backend.URL, false, nil)

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
		trackingPaths := []string{"/px.gif", "/collect", "/healthz", "/readyz", "/metrics", "/hmac.js", "/hmac/public-key"}

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

				router := NewMiddlewareRouter(mux, backend.URL, false, nil)

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

// TestNewMux tests mux creation
func TestNewMux(t *testing.T) {
	t.Run("creates mux without middleware mode", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				MiddlewareMode:     false,
				ForwardDestination: "",
			},
			Emit:    func(e event.Event) {},
			Metrics: metrics.InitMetrics(),
		}

		mux := NewMux(env)

		if mux == nil {
			t.Fatal("mux should not be nil")
		}
	})

	t.Run("creates middleware router when enabled", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		env := Env{
			Cfg: config.Config{
				MiddlewareMode:     true,
				ForwardDestination: backend.URL,
				AutoInjectPixel:    false,
			},
			Emit:    func(e event.Event) {},
			Metrics: metrics.InitMetrics(),
		}

		mux := NewMux(env)

		if mux == nil {
			t.Fatal("mux should not be nil")
		}

		// Test that it works as middleware
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should handle tracking paths
		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("disables middleware mode when destination is empty", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				MiddlewareMode:     true,
				ForwardDestination: "", // Empty destination
			},
			Emit:    func(e event.Event) {},
			Metrics: metrics.InitMetrics(),
		}

		mux := NewMux(env)

		if mux == nil {
			t.Fatal("mux should not be nil")
		}

		// Should still work as regular mux
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("disables middleware mode when destination is invalid", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				MiddlewareMode:     true,
				ForwardDestination: "://invalid-url",
			},
			Emit:    func(e event.Event) {},
			Metrics: metrics.InitMetrics(),
		}

		mux := NewMux(env)

		if mux == nil {
			t.Fatal("mux should not be nil")
		}
	})
}
