package httpx

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/shortontech/gotrack/internal/assets"
	event "github.com/shortontech/gotrack/internal/event"
	"github.com/shortontech/gotrack/internal/metrics"
	cfg "github.com/shortontech/gotrack/pkg/config"
)

var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00,
}

type Env struct {
	Cfg      cfg.Config        // <-- use cfg.Config here
	Emit     func(event.Event) // injected sink fan-out
	HMACAuth *HMACAuth         // HMAC authentication handler
	Metrics  *metrics.Metrics  // metrics collection
}

func (e Env) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (e Env) ServePixelJS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine which file to serve based on the path
	var content []byte
	switch r.URL.Path {
	case "/pixel.js", "/pixel.umd.js":
		content = assets.PixelUMDJS
	case "/pixel.esm.js":
		content = assets.PixelESMJS
	default:
		http.NotFound(w, r)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Header().Set("Access-Control-Allow-Origin", "*")      // Allow CORS for pixel script

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (e Env) Readyz(w http.ResponseWriter, r *http.Request) {
	// TODO: verify sink connectivity (Kafka/PG) before returning 200
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (e Env) HMACScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if e.HMACAuth == nil {
		http.Error(w, "HMAC authentication not configured", http.StatusNotFound)
		return
	}

	// Generate client-specific key for this IP using the request
	script := e.HMACAuth.GenerateClientScriptForRequest(r)
	if script == "" {
		http.Error(w, "HMAC client script not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate") // Don't cache - IP-specific
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(script))
}

func (e Env) HMACPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if e.HMACAuth == nil {
		http.Error(w, "HMAC authentication not configured", http.StatusNotFound)
		return
	}

	publicKey := e.HMACAuth.GetPublicKeyBase64()
	if publicKey == "" {
		http.Error(w, "HMAC public key not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"public_key": publicKey,
		"algorithm":  "HMAC-SHA256",
		"header":     "X-GoTrack-HMAC",
	})
}

func (e Env) Pixel(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Pixel handler called")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	evt := event.Event{Type: "pageview"}
	// We only set URL/query-derived attrs server-side; client device info comes via /collect.
	event.EnrichServerFields(r, &evt, e.Cfg)
	log.Printf("DEBUG: Event created, event_id=%s, type=%s", evt.EventID, evt.Type)
	if e.Emit != nil {
		log.Printf("DEBUG: Calling Emit function")
		e.Emit(evt)
		log.Printf("DEBUG: Emit returned")
	} else {
		log.Printf("DEBUG: ERROR - Emit is nil!")
	}
	writePixel(w, r.Method == http.MethodHead)
}

func writePixel(w http.ResponseWriter, headOnly bool) {
	h := w.Header()
	h.Set("Content-Type", "image/gif")
	h.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	h.Set("Pragma", "no-cache")
	h.Set("Expires", "0")
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pixelGIF)
}

// POST /collect — accepts a single Event object or an array of Events from JS.
func (e Env) Collect(w http.ResponseWriter, r *http.Request) {
	if !e.validateCollectRequest(w, r) {
		return
	}

	body, ok := e.readAndVerifyBody(w, r)
	if !ok {
		return
	}

	accepted, ok := e.processEvents(w, r, body)
	if !ok {
		return
	}

	e.sendCollectResponse(w, accepted)
}

func (e Env) validateCollectRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

func (e Env) readAndVerifyBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	defer r.Body.Close()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, e.Cfg.MaxBodyBytes))
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return nil, false
	}

	// Verify HMAC if authentication is enabled
	if e.HMACAuth != nil && !e.HMACAuth.VerifyHMAC(r, body) {
		http.Error(w, "invalid or missing HMAC signature", http.StatusUnauthorized)
		return nil, false
	}

	return body, true
}

func (e Env) processEvents(w http.ResponseWriter, r *http.Request, body []byte) (int, bool) {
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return 0, false
	}

	if len(raw) > 0 && raw[0] == '[' {
		return e.processEventArray(w, r, raw)
	}
	return e.processSingleEvent(w, r, raw)
}

func (e Env) processEventArray(w http.ResponseWriter, r *http.Request, raw json.RawMessage) (int, bool) {
	var arr []event.Event
	if err := json.Unmarshal(raw, &arr); err != nil {
		http.Error(w, "invalid json array", http.StatusBadRequest)
		return 0, false
	}
	for i := range arr {
		event.EnrichServerFields(r, &arr[i], e.Cfg)
		if e.Emit != nil {
			e.Emit(arr[i])
		}
	}
	return len(arr), true
}

func (e Env) processSingleEvent(w http.ResponseWriter, r *http.Request, raw json.RawMessage) (int, bool) {
	var ev event.Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		http.Error(w, "invalid json object", http.StatusBadRequest)
		return 0, false
	}
	event.EnrichServerFields(r, &ev, e.Cfg)

	// DEBUG: Log that we're about to emit
	log.Printf("DEBUG: Processing event type=%s, event_id=%s", ev.Type, ev.EventID)

	if e.Emit != nil {
		e.Emit(ev)
		log.Printf("DEBUG: Event emitted successfully")
	} else {
		log.Printf("DEBUG: ERROR - Emit function is nil!")
	}
	return 1, true
}

func (e Env) sendCollectResponse(w http.ResponseWriter, accepted int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Gotrack-Accepted", itoa(accepted))
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": accepted, "status": "ok"})
}

func itoa(i int) string { return fmtInt(i) }

// tiny int->string to avoid fmt import in this file
func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return sign + string(b[i:])
}
