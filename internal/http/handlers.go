package httpx

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	event "revinar.io/go.track/internal/event"
	cfg "revinar.io/go.track/pkg/config"
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
}

func (e Env) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
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
	
	script := e.HMACAuth.GenerateClientScript()
	if script == "" {
		http.Error(w, "HMAC client script not available", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if e.Cfg.DNTRespect && r.Header.Get("DNT") == "1" {
		writePixel(w, r.Method == http.MethodHead)
		return
	}
	evt := event.Event{Type: "pageview"}
	// We only set URL/query-derived attrs server-side; client device info comes via /collect.
	event.EnrichServerFields(r, &evt, e.Cfg)
	if e.Emit != nil {
		e.Emit(evt)
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	if e.Cfg.DNTRespect && r.Header.Get("DNT") == "1" {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": 0, "status": "dnt"})
		return
	}

	defer r.Body.Close()
	
	// Read the body for HMAC verification
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, e.Cfg.MaxBodyBytes))
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	
	// Verify HMAC if authentication is enabled
	if e.HMACAuth != nil && !e.HMACAuth.VerifyHMAC(r, body) {
		http.Error(w, "invalid or missing HMAC signature", http.StatusUnauthorized)
		return
	}
	
	// Parse the JSON
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	accepted := 0
	if len(raw) > 0 && raw[0] == '[' {
		var arr []event.Event
		if err := json.Unmarshal(raw, &arr); err != nil {
			http.Error(w, "invalid json array", http.StatusBadRequest)
			return
		}
		for i := range arr {
			event.EnrichServerFields(r, &arr[i], e.Cfg)
			if e.Emit != nil {
				e.Emit(arr[i])
			}
			accepted++
		}
	} else {
		var ev event.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			http.Error(w, "invalid json object", http.StatusBadRequest)
			return
		}
		event.EnrichServerFields(r, &ev, e.Cfg)
		if e.Emit != nil {
			e.Emit(ev)
		}
		accepted = 1
	}

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
