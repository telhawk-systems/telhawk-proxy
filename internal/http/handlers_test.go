package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shortontech/gotrack/internal/event"
	"github.com/shortontech/gotrack/internal/metrics"
	"github.com/shortontech/gotrack/pkg/config"
)

// TestHealthz tests the health check endpoint
func TestHealthz(t *testing.T) {
	t.Run("returns 200 OK", func(t *testing.T) {
		env := Env{}
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		env.Healthz(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if body != "ok" {
			t.Errorf("body = %q, want %q", body, "ok")
		}
	})

	t.Run("handles POST method", func(t *testing.T) {
		env := Env{}
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		w := httptest.NewRecorder()

		env.Healthz(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

// TestReadyz tests the readiness check endpoint
func TestReadyz(t *testing.T) {
	t.Run("returns 200 ready", func(t *testing.T) {
		env := Env{}
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		env.Readyz(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if body != "ready" {
			t.Errorf("body = %q, want %q", body, "ready")
		}
	})
}

// TestHMACScript tests the HMAC client script endpoint
func TestHMACScript(t *testing.T) {
	t.Run("returns 404 when HMAC not configured", func(t *testing.T) {
		env := Env{HMACAuth: nil}
		req := httptest.NewRequest(http.MethodGet, "/hmac.js", nil)
		w := httptest.NewRecorder()

		env.HMACScript(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("returns script when HMAC configured", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "")
		env := Env{HMACAuth: auth}
		req := httptest.NewRequest(http.MethodGet, "/hmac.js", nil)
		w := httptest.NewRecorder()

		env.HMACScript(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/javascript" {
			t.Errorf("Content-Type = %q, want application/javascript", contentType)
		}

		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "no-cache") {
			t.Errorf("Cache-Control should contain no-cache (IP-specific script), got %q", cacheControl)
		}

		body := w.Body.String()
		if len(body) == 0 {
			t.Error("script body should not be empty")
		}
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "")
		env := Env{HMACAuth: auth}
		req := httptest.NewRequest(http.MethodPost, "/hmac.js", nil)
		w := httptest.NewRecorder()

		env.HMACScript(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestHMACPublicKey tests the HMAC public key endpoint
func TestHMACPublicKey(t *testing.T) {
	t.Run("returns 404 when HMAC not configured", func(t *testing.T) {
		env := Env{HMACAuth: nil}
		req := httptest.NewRequest(http.MethodGet, "/hmac/public-key", nil)
		w := httptest.NewRecorder()

		env.HMACPublicKey(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("returns public key JSON when HMAC configured", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "")
		env := Env{HMACAuth: auth}
		req := httptest.NewRequest(http.MethodGet, "/hmac/public-key", nil)
		w := httptest.NewRecorder()

		env.HMACPublicKey(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}

		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "max-age=3600") {
			t.Errorf("Cache-Control should contain max-age=3600, got %q", cacheControl)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode JSON response: %v", err)
		}

		if result["public_key"] == "" {
			t.Error("public_key should not be empty")
		}
		if result["algorithm"] != "HMAC-SHA256" {
			t.Errorf("algorithm = %q, want HMAC-SHA256", result["algorithm"])
		}
		if result["header"] != "X-GoTrack-HMAC" {
			t.Errorf("header = %q, want X-GoTrack-HMAC", result["header"])
		}
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "")
		env := Env{HMACAuth: auth}
		req := httptest.NewRequest(http.MethodPost, "/hmac/public-key", nil)
		w := httptest.NewRecorder()

		env.HMACPublicKey(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestPixel tests the pixel tracking endpoint
func assertPixelResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatusCode int, expectBody bool) {
	t.Helper()
	if w.Code != wantStatusCode {
		t.Errorf("status code = %d, want %d", w.Code, wantStatusCode)
	}
	if wantStatusCode == http.StatusOK {
		if ct := w.Header().Get("Content-Type"); ct != "image/gif" {
			t.Errorf("Content-Type = %q, want image/gif", ct)
		}
		if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Errorf("Cache-Control should contain no-store, got %q", cc)
		}
		if expectBody && !bytes.Equal(w.Body.Bytes(), pixelGIF) {
			t.Error("response body should be pixel GIF")
		}
		if !expectBody && len(w.Body.Bytes()) > 0 {
			t.Error("HEAD request should not return body")
		}
	}
}

func TestPixel(t *testing.T) {
	t.Run("returns GIF for GET request", func(t *testing.T) {
		var emittedEvent *event.Event
		env := Env{Cfg: config.Config{}, Emit: func(e event.Event) { emittedEvent = &e }}
		req := httptest.NewRequest(http.MethodGet, "/px.gif?utm_source=test", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusOK, true)
		if emittedEvent == nil {
			t.Fatal("event should have been emitted")
		}
		if emittedEvent.Type != "pageview" {
			t.Errorf("event type = %q, want pageview", emittedEvent.Type)
		}
	})

	t.Run("returns GIF for HEAD request without body", func(t *testing.T) {
		env := Env{Cfg: config.Config{}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodHead, "/px.gif", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusOK, false)
	})

	t.Run("rejects invalid methods", func(t *testing.T) {
		env := Env{Cfg: config.Config{}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/px.gif", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusMethodNotAllowed, false)
	})

	t.Run("handles nil Emit gracefully", func(t *testing.T) {
		env := Env{Cfg: config.Config{}, Emit: nil}
		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusOK, true)
	})
}

// TestCollect tests the event collection endpoint
func assertResponseStatus(t *testing.T, w *httptest.ResponseRecorder, wantCode int, wantContentType string) {
	t.Helper()
	if w.Code != wantCode {
		t.Errorf("status code = %d, want %d", w.Code, wantCode)
	}
	if wantContentType != "" {
		if got := w.Header().Get("Content-Type"); got != wantContentType {
			t.Errorf("Content-Type = %q, want %q", got, wantContentType)
		}
	}
}

func assertAcceptedCount(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if got := w.Header().Get("X-Gotrack-Accepted"); got != fmt.Sprintf("%d", want) {
		t.Errorf("X-Gotrack-Accepted = %q, want %d", got, want)
	}
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["accepted"] != float64(want) {
		t.Errorf("accepted = %v, want %d", response["accepted"], want)
	}
}

// TestWritePixel tests the pixel writing helper
func TestWritePixel(t *testing.T) {
	t.Run("writes pixel for normal request", func(t *testing.T) {
		w := httptest.NewRecorder()
		writePixel(w, false)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "image/gif" {
			t.Errorf("Content-Type = %q, want image/gif", contentType)
		}

		body := w.Body.Bytes()
		if !bytes.Equal(body, pixelGIF) {
			t.Error("body should match pixelGIF")
		}
	})

	t.Run("writes no body for HEAD request", func(t *testing.T) {
		w := httptest.NewRecorder()
		writePixel(w, true)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.Bytes()
		if len(body) > 0 {
			t.Error("HEAD request should not have body")
		}

		// Headers should still be set
		contentType := w.Header().Get("Content-Type")
		if contentType != "image/gif" {
			t.Errorf("Content-Type = %q, want image/gif", contentType)
		}
	})

	t.Run("sets proper cache headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		writePixel(w, false)

		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "no-store") {
			t.Errorf("Cache-Control should contain no-store")
		}

		pragma := w.Header().Get("Pragma")
		if pragma != "no-cache" {
			t.Errorf("Pragma = %q, want no-cache", pragma)
		}

		expires := w.Header().Get("Expires")
		if expires != "0" {
			t.Errorf("Expires = %q, want 0", expires)
		}
	})
}

// TestFmtInt tests the integer formatting utility
func TestFmtInt(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-1, "-1"},
		{-123, "-123"},
		{9999, "9999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := fmtInt(tt.input)
			if got != tt.want {
				t.Errorf("fmtInt(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestItoa tests the itoa wrapper
func TestItoa(t *testing.T) {
	if itoa(42) != "42" {
		t.Errorf("itoa(42) = %q, want 42", itoa(42))
	}
	if itoa(0) != "0" {
		t.Errorf("itoa(0) = %q, want 0", itoa(0))
	}
}

// TestCollectIntegration tests realistic end-to-end scenarios
func TestCollectIntegration(t *testing.T) {
	t.Run("full event with enrichment", func(t *testing.T) {
		var capturedEvent *event.Event
		env := Env{
			Cfg: config.Config{
				MaxBodyBytes: 1024 * 1024,
				TrustProxy:   false,
			},
			Emit: func(e event.Event) {
				capturedEvent = &e
			},
			Metrics: metrics.InitMetrics(),
		}

		eventJSON := `{
			"type": "click",
			"event_id": "test-event-123"
		}`

		req := httptest.NewRequest(http.MethodPost, "/collect?utm_source=test_source", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TestBot/1.0")
		req.Header.Set("Referer", "https://example.com/page")
		req.RemoteAddr = "203.0.113.42:12345"

		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}

		if capturedEvent == nil {
			t.Fatal("event should have been captured")
		}

		// Verify enrichment happened
		if capturedEvent.EventID != "test-event-123" {
			t.Errorf("event_id = %q, want test-event-123", capturedEvent.EventID)
		}

		if capturedEvent.Type != "click" {
			t.Errorf("type = %q, want click", capturedEvent.Type)
		}

		// Server should enrich user agent
		if capturedEvent.Device.UA != "TestBot/1.0" {
			t.Errorf("user agent = %q, want TestBot/1.0", capturedEvent.Device.UA)
		}

		// Server should enrich referrer
		if capturedEvent.URL.Referrer != "https://example.com/page" {
			t.Errorf("referrer = %q, want https://example.com/page", capturedEvent.URL.Referrer)
		}

		// Server should enrich IP
		if capturedEvent.Server.IP != "203.0.113.42" {
			t.Errorf("IP = %q, want 203.0.113.42", capturedEvent.Server.IP)
		}
	})
}

// TestServePixelJS tests the pixel JS file serving endpoint
func TestServePixelJS(t *testing.T) {
	// Create a temporary test file
	env := Env{}

	t.Run("serves embedded pixel.js", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pixel.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		if len(w.Body.Bytes()) == 0 {
			t.Error("expected non-empty response body")
		}
	})

	t.Run("returns 404 for unknown paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("rejects POST method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/pixel.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("supports HEAD method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/pixel.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		// Will be 404 since file doesn't exist, but should not be method not allowed
		if w.Code == http.StatusMethodNotAllowed {
			t.Errorf("HEAD method should be allowed")
		}
	})

	t.Run("sets correct headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pixel.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		// Even on 404, these headers should be set if we get past method check
		contentType := w.Header().Get("Content-Type")
		if w.Code == http.StatusOK && contentType != "application/javascript" {
			t.Errorf("Content-Type = %q, want application/javascript", contentType)
		}
	})
}

// Test ServePixelJS with all paths
func TestServePixelJS_ComprehensivePaths(t *testing.T) {
	env := Env{}

	tests := []struct {
		name         string
		method       string
		path         string
		wantStatus   int
		checkContent bool
	}{
		{
			name:         "GET /pixel.js",
			method:       "GET",
			path:         "/pixel.js",
			wantStatus:   200,
			checkContent: true,
		},
		{
			name:         "GET /pixel.umd.js",
			method:       "GET",
			path:         "/pixel.umd.js",
			wantStatus:   200,
			checkContent: true,
		},
		{
			name:         "GET /pixel.esm.js",
			method:       "GET",
			path:         "/pixel.esm.js",
			wantStatus:   200,
			checkContent: true,
		},
		{
			name:       "HEAD /pixel.js",
			method:     "HEAD",
			path:       "/pixel.js",
			wantStatus: 200,
		},
		{
			name:       "HEAD /pixel.esm.js",
			method:     "HEAD",
			path:       "/pixel.esm.js",
			wantStatus: 200,
		},
		{
			name:       "POST not allowed",
			method:     "POST",
			path:       "/pixel.js",
			wantStatus: 405,
		},
		{
			name:       "PUT not allowed",
			method:     "PUT",
			path:       "/pixel.js",
			wantStatus: 405,
		},
		{
			name:       "DELETE not allowed",
			method:     "DELETE",
			path:       "/pixel.js",
			wantStatus: 405,
		},
		{
			name:       "Unknown path",
			method:     "GET",
			path:       "/unknown.js",
			wantStatus: 404,
		},
		{
			name:       "Root path",
			method:     "GET",
			path:       "/",
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			env.ServePixelJS(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == 200 {
				ct := w.Header().Get("Content-Type")
				if ct != "application/javascript" {
					t.Errorf("Content-Type = %q, want application/javascript", ct)
				}

				cc := w.Header().Get("Cache-Control")
				if cc != "public, max-age=3600" {
					t.Errorf("Cache-Control = %q, want public, max-age=3600", cc)
				}

				cors := w.Header().Get("Access-Control-Allow-Origin")
				if cors != "*" {
					t.Errorf("CORS = %q, want *", cors)
				}

				if tt.checkContent && tt.method == "GET" {
					if len(w.Body.Bytes()) == 0 {
						t.Error("expected non-empty response body for embedded asset")
					}
				}

				if tt.method == "HEAD" && len(w.Body.Bytes()) != 0 {
					t.Error("HEAD should not return body")
				}
			}
		})
	}
}

// Test ServePixelJS with embedded assets (no file dependencies)
func TestServePixelJS_MissingFiles(t *testing.T) {
	env := Env{}

	req := httptest.NewRequest("GET", "/pixel.js", nil)
	w := httptest.NewRecorder()

	env.ServePixelJS(w, req)

	// Should return 200 because assets are embedded, not file-based
	if w.Code != 200 {
		t.Errorf("status = %d, want 200 (embedded assets don't depend on files)", w.Code)
	}

	if len(w.Body.Bytes()) == 0 {
		t.Error("expected non-empty response body from embedded asset")
	}
}
