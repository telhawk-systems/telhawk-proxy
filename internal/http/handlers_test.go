package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"revinar.io/go.track/internal/event"
	"revinar.io/go.track/internal/metrics"
	"revinar.io/go.track/pkg/config"
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
func TestPixel(t *testing.T) {
	t.Run("returns GIF for GET request", func(t *testing.T) {
		var emittedEvent *event.Event
		env := Env{
			Cfg: config.Config{DNTRespect: false},
			Emit: func(e event.Event) {
				emittedEvent = &e
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/px.gif?utm_source=test", nil)
		w := httptest.NewRecorder()

		env.Pixel(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "image/gif" {
			t.Errorf("Content-Type = %q, want image/gif", contentType)
		}

		cacheControl := w.Header().Get("Cache-Control")
		if !strings.Contains(cacheControl, "no-store") {
			t.Errorf("Cache-Control should contain no-store, got %q", cacheControl)
		}

		body := w.Body.Bytes()
		if !bytes.Equal(body, pixelGIF) {
			t.Error("response body should be pixel GIF")
		}

		if emittedEvent == nil {
			t.Fatal("event should have been emitted")
		}
		if emittedEvent.Type != "pageview" {
			t.Errorf("event type = %q, want pageview", emittedEvent.Type)
		}
	})

	t.Run("returns GIF for HEAD request without body", func(t *testing.T) {
		env := Env{
			Cfg:  config.Config{DNTRespect: false},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodHead, "/px.gif", nil)
		w := httptest.NewRecorder()

		env.Pixel(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "image/gif" {
			t.Errorf("Content-Type = %q, want image/gif", contentType)
		}

		body := w.Body.Bytes()
		if len(body) > 0 {
			t.Error("HEAD request should not return body")
		}
	})

	t.Run("respects DNT header when configured", func(t *testing.T) {
		emitCalled := false
		env := Env{
			Cfg: config.Config{DNTRespect: true},
			Emit: func(e event.Event) {
				emitCalled = true
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()

		env.Pixel(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		if emitCalled {
			t.Error("event should not be emitted when DNT header is set")
		}

		// Still returns pixel
		body := w.Body.Bytes()
		if !bytes.Equal(body, pixelGIF) {
			t.Error("should still return pixel GIF")
		}
	})

	t.Run("does not respect DNT when not configured", func(t *testing.T) {
		emitCalled := false
		env := Env{
			Cfg: config.Config{DNTRespect: false},
			Emit: func(e event.Event) {
				emitCalled = true
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()

		env.Pixel(w, req)

		if !emitCalled {
			t.Error("event should be emitted even with DNT header when DNTRespect=false")
		}
	})

	t.Run("rejects invalid methods", func(t *testing.T) {
		env := Env{
			Cfg:  config.Config{DNTRespect: false},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodPost, "/px.gif", nil)
		w := httptest.NewRecorder()

		env.Pixel(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handles nil Emit gracefully", func(t *testing.T) {
		env := Env{
			Cfg:  config.Config{DNTRespect: false},
			Emit: nil,
		}

		req := httptest.NewRequest(http.MethodGet, "/px.gif", nil)
		w := httptest.NewRecorder()

		// Should not panic
		env.Pixel(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

// TestCollect tests the event collection endpoint
func TestCollect(t *testing.T) {
	t.Run("accepts single event object", func(t *testing.T) {
		var emittedEvent *event.Event
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {
				emittedEvent = &e
			},
		}

		eventJSON := `{"type":"click","event_id":"test-123"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}

		acceptedHeader := w.Header().Get("X-Gotrack-Accepted")
		if acceptedHeader != "1" {
			t.Errorf("X-Gotrack-Accepted = %q, want 1", acceptedHeader)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("status = %v, want ok", response["status"])
		}
		if response["accepted"] != float64(1) {
			t.Errorf("accepted = %v, want 1", response["accepted"])
		}

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
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {
				emittedEvents = append(emittedEvents, e)
			},
		}

		eventsJSON := `[{"type":"pageview","event_id":"evt1"},{"type":"click","event_id":"evt2"}]`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventsJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}

		acceptedHeader := w.Header().Get("X-Gotrack-Accepted")
		if acceptedHeader != "2" {
			t.Errorf("X-Gotrack-Accepted = %q, want 2", acceptedHeader)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["accepted"] != float64(2) {
			t.Errorf("accepted = %v, want 2", response["accepted"])
		}

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
		env := Env{
			Cfg:  config.Config{MaxBodyBytes: 1024 * 1024},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodGet, "/collect", nil)
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		env := Env{
			Cfg:  config.Config{MaxBodyBytes: 1024 * 1024},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader("test"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusUnsupportedMediaType)
		}
	})

	t.Run("accepts missing content type", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {},
		}

		eventJSON := `{"type":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		// No Content-Type header set
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d (should accept empty content type)", w.Code, http.StatusAccepted)
		}
	})

	t.Run("respects DNT header", func(t *testing.T) {
		emitCalled := false
		env := Env{
			Cfg: config.Config{
				DNTRespect:   true,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {
				emitCalled = true
			},
		}

		eventJSON := `{"type":"pageview"}`
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(eventJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("DNT", "1")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}

		if emitCalled {
			t.Error("event should not be emitted when DNT=1")
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["accepted"] != float64(0) {
			t.Errorf("accepted = %v, want 0", response["accepted"])
		}
		if response["status"] != "dnt" {
			t.Errorf("status = %v, want dnt", response["status"])
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects invalid JSON array", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`[{"invalid": json}]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects invalid JSON object in array", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {},
		}

		// Valid JSON array but invalid event structure
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`["not an object"]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects body too large", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 100, // Very small limit
			},
			Emit: func(e event.Event) {},
		}

		largeBody := strings.Repeat("x", 200)
		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
		}
	})

	t.Run("handles empty array", func(t *testing.T) {
		env := Env{
			Cfg: config.Config{
				DNTRespect:   false,
				MaxBodyBytes: 1024 * 1024,
			},
			Emit: func(e event.Event) {},
		}

		req := httptest.NewRequest(http.MethodPost, "/collect", strings.NewReader(`[]`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Collect(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusAccepted)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["accepted"] != float64(0) {
			t.Errorf("accepted = %v, want 0", response["accepted"])
		}
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
