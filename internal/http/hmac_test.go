package httpx

import (
	"bytes"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHMACAuth(t *testing.T) {
	t.Run("creates auth with secret", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false)
		if auth == nil {
			t.Fatal("NewHMACAuth returned nil")
		}
		if !bytes.Equal(auth.secret, []byte("test-secret")) {
			t.Errorf("secret not set correctly")
		}
	})

	t.Run("derives public key when not provided", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false)
		if len(auth.publicKey) == 0 {
			t.Error("public key should be derived")
		}
		if auth.GetPublicKeyBase64() == "" {
			t.Error("GetPublicKeyBase64 should return non-empty string")
		}
	})

	t.Run("uses provided public key", func(t *testing.T) {
		providedKey := base64.StdEncoding.EncodeToString([]byte("custom-public-key"))
		auth := NewHMACAuth("test-secret", providedKey, false)
		if !bytes.Equal(auth.publicKey, []byte("custom-public-key")) {
			t.Errorf("should use provided public key")
		}
	})

	t.Run("falls back to derived key on invalid base64", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "not-valid-base64!!!", false)
		if len(auth.publicKey) == 0 {
			t.Error("should derive key when provided key is invalid")
		}
	})
}

func TestDerivePublicKey(t *testing.T) {
	auth := NewHMACAuth("test-secret", "", false)

	t.Run("derives 16-byte key", func(t *testing.T) {
		key := auth.derivePublicKey([]byte("test"))
		if len(key) != 16 {
			t.Errorf("derived key length = %d, want 16", len(key))
		}
	})
}

// Note: normalizeIP and getClientIP are internal functions tested indirectly

// Note: generateHMAC is an internal method tested indirectly through VerifyHMAC

func TestVerifyHMAC(t *testing.T) {
	secret := "test-secret"
	auth := NewHMACAuth(secret, "", true) // requireHMAC = true
	payload := []byte(`{"test":"data"}`)

	t.Run("rejects missing HMAC header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "192.168.1.1:8080"

		if auth.VerifyHMAC(req, payload) {
			t.Error("should reject missing HMAC header")
		}
	})

	t.Run("rejects invalid HMAC", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set("X-GoTrack-HMAC", "invalid-hmac-value")

		if auth.VerifyHMAC(req, payload) {
			t.Error("should reject invalid HMAC")
		}
	})

	t.Run("accepts when HMAC not required", func(t *testing.T) {
		authOptional := NewHMACAuth(secret, "", false) // requireHMAC = false
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "192.168.1.1:8080"

		if !authOptional.VerifyHMAC(req, payload) {
			t.Error("should accept when HMAC not required")
		}
	})

	t.Run("rejects when secret not configured", func(t *testing.T) {
		authNoSecret := NewHMACAuth("", "", true) // requireHMAC = true, no secret
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "192.168.1.1:8080"
		req.Header.Set("X-GoTrack-HMAC", "some-hmac")

		if authNoSecret.VerifyHMAC(req, payload) {
			t.Error("should reject when no secret configured")
		}
	})
}

func TestGenerateClientScript(t *testing.T) {
	t.Run("generates script with public key", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false)
		script := auth.GenerateClientScript()

		if script == "" {
			t.Error("should generate non-empty script")
		}
		if !strings.Contains(script, "GOTRACK_PUBLIC_KEY") {
			t.Error("script should contain public key constant")
		}
		if !strings.Contains(script, "generateHMAC") {
			t.Error("script should contain generateHMAC function")
		}
		if !strings.Contains(script, "X-GoTrack-HMAC") {
			t.Error("script should set X-GoTrack-HMAC header")
		}
	})

	t.Run("returns empty when no public key", func(t *testing.T) {
		auth := &HMACAuth{} // Empty auth with no keys
		script := auth.GenerateClientScript()
		if script != "" {
			t.Error("should return empty script when no public key")
		}
	})

	t.Run("includes base64 public key in script", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "", false)
		publicKeyB64 := auth.GetPublicKeyBase64()
		script := auth.GenerateClientScript()

		if !strings.Contains(script, publicKeyB64) {
			t.Error("script should contain base64 public key")
		}
	})
}

// TestGetClientIP tests IP extraction from requests
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xForwarded string
		xRealIP    string
		want       string
	}{
		{
			name:       "gets IP from RemoteAddr when no headers",
			remoteAddr: "203.0.113.42:12345",
			want:       "203.0.113.42:12345",
		},
		{
			name:       "prefers X-Forwarded-For over X-Real-IP",
			remoteAddr: "192.168.1.1:8080",
			xRealIP:    "10.0.0.1",
			xForwarded: "203.0.113.42",
			want:       "203.0.113.42",
		},
		{
			name:       "prefers X-Real-IP over RemoteAddr",
			remoteAddr: "192.168.1.1:8080",
			xRealIP:    "203.0.113.42",
			want:       "203.0.113.42",
		},
		{
			name:       "uses first IP in X-Forwarded-For chain",
			remoteAddr: "192.168.1.1:8080",
			xForwarded: "203.0.113.42, 10.0.0.1, 192.168.1.1",
			want:       "203.0.113.42",
		},
		{
			name:       "handles IPv6 RemoteAddr",
			remoteAddr: "[2001:db8::1]:8080",
			want:       "[2001:db8::1]:8080",
		},
		{
			name:       "handles RemoteAddr without port",
			remoteAddr: "203.0.113.42",
			want:       "203.0.113.42",
		},
		{
			name:       "trims whitespace from X-Forwarded-For",
			remoteAddr: "192.168.1.1:8080",
			xForwarded: "  203.0.113.42  , 10.0.0.1",
			want:       "203.0.113.42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNormalizeIP tests IP normalization (port removal)
func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "strips port from IPv4",
			ip:   "203.0.113.42:12345",
			want: "203.0.113.42",
		},
		{
			name: "strips port from IPv4 with standard port",
			ip:   "192.168.1.100:80",
			want: "192.168.1.100",
		},
		{
			name: "strips port from IPv6",
			ip:   "[2001:db8:85a3::8a2e:370:7334]:8080",
			want: "2001:db8:85a3::8a2e:370:7334",
		},
		{
			name: "handles IPv6 without brackets",
			ip:   "2001:db8::1",
			want: "2001:db8::1",
		},
		{
			name: "handles IPv4 without port",
			ip:   "203.0.113.42",
			want: "203.0.113.42",
		},
		{
			name: "handles empty string",
			ip:   "",
			want: "",
		},
		{
			name: "handles malformed input",
			ip:   "not-an-ip",
			want: "not-an-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIP(tt.ip)
			if got != tt.want {
				t.Errorf("normalizeIP(%q) = %q, want %q", tt.ip, got, tt.want)
			}
		})
	}
}

// Note: deriveClientKey is an internal method tested indirectly

func TestHMACIntegration(t *testing.T) {
	t.Run("handles optional HMAC gracefully", func(t *testing.T) {
		secret := "integration-test-secret"
		auth := NewHMACAuth(secret, "", false) // HMAC not required
		payload := []byte(`{"event":"test","data":"value"}`)

		// Request without HMAC should still work
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "203.0.113.1:12345"

		if !auth.VerifyHMAC(req, payload) {
			t.Error("should accept request when HMAC not required")
		}
	})
}
