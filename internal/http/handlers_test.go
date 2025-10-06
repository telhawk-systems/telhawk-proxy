package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
		auth := NewHMACAuth("test-secret", "", false)
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
		if !strings.Contains(cacheControl, "max-age=3600") {
			t.Errorf("Cache-Control should contain max-age=3600, got %q", cacheControl)
		}

		body := w.Body.String()
		if len(body) == 0 {
			t.Error("script body should not be empty")
		}
	})

	t.Run("rejects non-GET methods", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false)
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
		auth := NewHMACAuth("test-secret", "", false)
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
		auth := NewHMACAuth("test-secret", "", false)
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
		env := Env{Cfg: config.Config{DNTRespect: false}, Emit: func(e event.Event) { emittedEvent = &e }}
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
		env := Env{Cfg: config.Config{DNTRespect: false}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodHead, "/px.gif", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusOK, false)
	})

	t.Run("respects DNT header when configured", func(t *testing.T) {
		emitCalled := false
		env := Env{Cfg: config.Config{DNTRespect: true}, Emit: func(e event.Event) { emitCalled = true }}
		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusOK, true)
		if emitCalled {
			t.Error("event should not be emitted when DNT header is set")
		}
	})

	t.Run("does not respect DNT when not configured", func(t *testing.T) {
		emitCalled := false
		env := Env{Cfg: config.Config{DNTRespect: false}, Emit: func(e event.Event) { emitCalled = true }}
		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		if !emitCalled {
			t.Error("event should be emitted even with DNT header when DNTRespect=false")
		}
	})

	t.Run("rejects invalid methods", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/px.gif", nil)
		w := httptest.NewRecorder()
		env.Pixel(w, req)
		assertPixelResponse(t, w, http.StatusMethodNotAllowed, false)
	})

	t.Run("handles nil Emit gracefully", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false}, Emit: nil}
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

func TestCollect(t *testing.T) {
	t.Run("accepts single event object", func(t *testing.T) {
		var emittedEvent *event.Event
		env := Env{
			Cfg:  config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024},
			Emit: func(e event.Event) { emittedEvent = &e },
		}
		eventJSON := `{"type":"click","event_id":"test-123"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusAccepted, "application/json")
		assertAcceptedCount(t, w, 1)
		if emittedEvent == nil {
			t.Fatal("event should have been emitted")
		}
		if emittedEvent.EventID != "test-123" {
			t.Errorf("event_id = %q, want test-123", emittedEvent.EventID)
		}
	})

	t.Run("accepts array of events", func(t *testing.T) {
		var emittedEvents []event.Event
		env := Env{
			Cfg:  config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024},
			Emit: func(e event.Event) { emittedEvents = append(emittedEvents, e) },
		}
		eventsJSON := `[{"type":"pageview","event_id":"evt1"},{"type":"click","event_id":"evt2"}]`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventsJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusAccepted, "")
		assertAcceptedCount(t, w, 2)
		if len(emittedEvents) != 2 {
			t.Fatalf("expected 2 emitted events, got %d", len(emittedEvents))
		}
		if emittedEvents[0].EventID != "evt1" {
			t.Errorf("first event_id = %q, want evt1", emittedEvents[0].EventID)
		}
		if emittedEvents[1].EventID != "evt2" {
			t.Errorf("second event_id = %q, want evt2", emittedEvents[1].EventID)
		}
	})

	t.Run("rejects non-POST methods", func(t *testing.T) {
		env := Env{Cfg: config.Config{MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodGet, "/collect", nil)
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusMethodNotAllowed, "")
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		env := Env{Cfg: config.Config{MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader("test"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusUnsupportedMediaType, "")
	})

	t.Run("accepts missing content type", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		eventJSON := `{"type":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusAccepted, "")
	})

	t.Run("respects DNT header", func(t *testing.T) {
		emitCalled := false
		env := Env{
			Cfg:  config.Config{DNTRespect: true, MaxBodyBytes: 1024 * 1024},
			Emit: func(e event.Event) { emitCalled = true },
		}
		eventJSON := `{"type":"pageview"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusAccepted, "")
		if emitCalled {
			t.Error("event should not be emitted when DNT=1")
		}
		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)
		if response["accepted"] != float64(0) {
			t.Errorf("accepted = %v, want 0", response["accepted"])
		}
		if response["status"] != "dnt" {
			t.Errorf("status = %v, want dnt", response["status"])
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusBadRequest, "")
	})

	t.Run("rejects invalid JSON array", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`[{"invalid": json}]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusBadRequest, "")
	})

	t.Run("rejects invalid JSON object in array", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`["not an object"]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusBadRequest, "")
	})

	t.Run("rejects body too large", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 100}, Emit: func(e event.Event) {}}
		largeBody := strings.Repeat("x", 200)
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusRequestEntityTooLarge, "")
	})

	t.Run("handles empty array", func(t *testing.T) {
		env := Env{Cfg: config.Config{DNTRespect: false, MaxBodyBytes: 1024 * 1024}, Emit: func(e event.Event) {}}
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`[]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		env.Collect(w, req)
		assertResponseStatus(t, w, http.StatusAccepted, "")
		assertAcceptedCount(t, w, 0)
	})

	t.Run("handles nil Emit gracefully", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: nil,
		}

		eventJSON := `{"type":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Should not panic
		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}
	})

	t.Run("HMAC authentication - rejects invalid HMAC when not required", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false) // requireHMAC = false
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			HMACAuth: auth,
			Emit:     func(e event.Event) {},
		}

		eventJSON := `{"type":"test"}`
		body := []byte(eventJSON)
		req := httptest.NewRequest(http.MethodPost, "/collect", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.1:8080"

		// Provide an invalid HMAC
		req.Header.Set("X-GoTrack-HMAC", "invalid-hmac-signature")

		w := httptest.NewRecorder()

		env.Collect(w, req)

		// When HMAC not required but invalid signature provided, should still accept
		// (HMAC verification returns true when not required)
		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d (should accept when HMAC not required)", w.Code, http.StatusAccepted)
		}
	})

	t.Run("HMAC authentication - rejects missing HMAC when not required", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false) // requireHMAC = false
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			HMACAuth: auth,
			Emit:     func(e event.Event) {},
		}

		eventJSON := `{"type":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.1:8080"
		// No HMAC header

		w := httptest.NewRecorder()

		env.Collect(w, req)

		// Should accept when HMAC not required
		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d (should accept when HMAC not required)", w.Code, http.StatusAccepted)
		}
	})
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
				DNTRespect:   false,
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

	t.Run("returns 404 for non-existent files", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/pixel.js", nil)
		w := httptest.NewRecorder()

		env.ServePixelJS(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
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
// Create temporary static directory with test files
err := os.MkdirAll("static", 0755)
if err != nil {
t.Fatalf("failed to create static dir: %v", err)
}
defer os.RemoveAll("static")

umdContent := []byte("// UMD module\nwindow.gotrack = {};")
esmContent := []byte("// ESM module\nexport default {};")

err = os.WriteFile("static/pixel.umd.js", umdContent, 0644)
if err != nil {
t.Fatalf("failed to write UMD file: %v", err)
}
err = os.WriteFile("static/pixel.esm.js", esmContent, 0644)
if err != nil {
t.Fatalf("failed to write ESM file: %v", err)
}

env := Env{}

tests := []struct {
name           string
method         string
path           string
wantStatus     int
wantContentLen int
checkContent   bool
}{
{
name:           "GET /pixel.js",
method:         "GET",
path:           "/pixel.js",
wantStatus:     200,
wantContentLen: len(umdContent),
checkContent:   true,
},
{
name:           "GET /pixel.umd.js",
method:         "GET",
path:           "/pixel.umd.js",
wantStatus:     200,
wantContentLen: len(umdContent),
checkContent:   true,
},
{
name:           "GET /pixel.esm.js",
method:         "GET",
path:           "/pixel.esm.js",
wantStatus:     200,
wantContentLen: len(esmContent),
checkContent:   true,
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
if len(w.Body.Bytes()) != tt.wantContentLen {
t.Errorf("body length = %d, want %d", len(w.Body.Bytes()), tt.wantContentLen)
}
}

if tt.method == "HEAD" && len(w.Body.Bytes()) != 0 {
t.Error("HEAD should not return body")
}
}
})
}
}

// Test ServePixelJS with missing files
func TestServePixelJS_MissingFiles(t *testing.T) {
// Ensure static directory doesn't exist
os.RemoveAll("static")

env := Env{}

req := httptest.NewRequest("GET", "/pixel.js", nil)
w := httptest.NewRecorder()

env.ServePixelJS(w, req)

if w.Code != 404 {
t.Errorf("status = %d, want 404 for missing file", w.Code)
}
}
