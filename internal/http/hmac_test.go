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
		auth := NewHMACAuth("test-secret", "")
		if auth == nil {
			t.Fatal("NewHMACAuth returned nil")
		}
		if !bytes.Equal(auth.secret, []byte("test-secret")) {
			t.Errorf("secret not set correctly")
		}
	})

	t.Run("derives public key when not provided", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "")
		if len(auth.publicKey) == 0 {
			t.Error("public key should be derived")
		}
		if auth.GetPublicKeyBase64() == "" {
			t.Error("GetPublicKeyBase64 should return non-empty string")
		}
	})

	t.Run("uses provided public key", func(t *testing.T) {
		providedKey := base64.StdEncoding.EncodeToString([]byte("custom-public-key"))
		auth := NewHMACAuth("test-secret", providedKey)
		if !bytes.Equal(auth.publicKey, []byte("custom-public-key")) {
			t.Errorf("should use provided public key")
		}
	})

	t.Run("falls back to derived key on invalid base64", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "not-valid-base64!!!")
		if len(auth.publicKey) == 0 {
			t.Error("should derive key when provided key is invalid")
		}
	})
}

func TestDerivePublicKey(t *testing.T) {
	auth := NewHMACAuth("test-secret", "")

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
	auth := NewHMACAuth(secret, "")
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

	t.Run("rejects when HMAC not provided", func(t *testing.T) {
		auth := NewHMACAuth(secret, "")
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "192.168.1.1:8080"

		// HMAC is required when auth is configured
		if auth.VerifyHMAC(req, payload) {
			t.Error("should reject when HMAC header is missing")
		}
	})

	t.Run("rejects when secret not configured", func(t *testing.T) {
		authNoSecret := NewHMACAuth("", "") // requireHMAC = true, no secret
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
		auth := NewHMACAuth("test-secret", "")
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
		auth := NewHMACAuth("test-secret", "")
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
	t.Run("rejects request without HMAC", func(t *testing.T) {
		secret := "integration-test-secret"
		auth := NewHMACAuth(secret, "")
		payload := []byte(`{"event":"test","data":"value"}`)

		// Request without HMAC should be rejected when auth is configured
		req := httptest.NewRequest("POST", "/collect", bytes.NewReader(payload))
		req.RemoteAddr = "203.0.113.1:12345"

		if auth.VerifyHMAC(req, payload) {
			t.Error("should reject request when HMAC is missing")
		}
	})
}

// Test GetPublicKeyBase64 with various states
func TestHMACAuth_GetPublicKeyBase64_Comprehensive(t *testing.T) {
	t.Run("with valid public key", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public-key")

		pubKey := auth.GetPublicKeyBase64()
		if pubKey == "" {
			t.Error("public key should not be empty")
		}

		// Verify it's valid base64
		decoded, err := base64.StdEncoding.DecodeString(pubKey)
		if err != nil {
			t.Errorf("public key should be valid base64: %v", err)
		}
		if len(decoded) == 0 {
			t.Error("decoded public key should not be empty")
		}
	})

	t.Run("with empty public key", func(t *testing.T) {
		auth := &HMACAuth{
			publicKey: []byte{},
		}

		pubKey := auth.GetPublicKeyBase64()
		if pubKey != "" {
			t.Errorf("public key should be empty, got %q", pubKey)
		}
	})

	t.Run("with nil public key", func(t *testing.T) {
		auth := &HMACAuth{
			publicKey: nil,
		}

		pubKey := auth.GetPublicKeyBase64()
		if pubKey != "" {
			t.Errorf("public key should be empty, got %q", pubKey)
		}
	})

	t.Run("derived key format", func(t *testing.T) {
		auth := NewHMACAuth("my-secret-key", "")

		pubKey := auth.GetPublicKeyBase64()
		if pubKey == "" {
			t.Error("derived public key should not be empty")
		}

		// Verify it decodes correctly
		decoded, err := base64.StdEncoding.DecodeString(pubKey)
		if err != nil {
			t.Errorf("derived key should be valid base64: %v", err)
		}

		// Should be 16 bytes (first 16 bytes of HMAC)
		if len(decoded) != 16 {
			t.Errorf("derived key length = %d, want 16", len(decoded))
		}
	})
}

// Test generateHMAC with edge cases
func TestHMACAuth_GenerateHMAC_EdgeCases(t *testing.T) {
	t.Run("with empty secret", func(t *testing.T) {
		auth := &HMACAuth{
			secret: []byte{},
		}

		hmac := auth.generateHMAC([]byte("test payload"), "127.0.0.1")
		if hmac != "" {
			t.Error("HMAC should be empty when secret is empty")
		}
	})

	t.Run("with nil secret", func(t *testing.T) {
		auth := &HMACAuth{
			secret: nil,
		}

		hmac := auth.generateHMAC([]byte("test payload"), "127.0.0.1")
		if hmac != "" {
			t.Error("HMAC should be empty when secret is nil")
		}
	})

	t.Run("with different IPs produce different HMACs", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")
		payload := []byte(`{"event":"click"}`)

		hmac1 := auth.generateHMAC(payload, "192.168.1.1")
		hmac2 := auth.generateHMAC(payload, "192.168.1.2")

		if hmac1 == hmac2 {
			t.Error("different IPs should produce different HMACs")
		}

		if hmac1 == "" || hmac2 == "" {
			t.Error("HMACs should not be empty")
		}
	})

	t.Run("with IPv6 addresses", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")
		payload := []byte(`{"event":"click"}`)

		hmac := auth.generateHMAC(payload, "2001:0db8:85a3:0000:0000:8a2e:0370:7334")
		if hmac == "" {
			t.Error("HMAC should be generated for IPv6")
		}
	})

	t.Run("with IP and port", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")
		payload := []byte(`{"event":"click"}`)

		hmac := auth.generateHMAC(payload, "192.168.1.1:8080")
		if hmac == "" {
			t.Error("HMAC should be generated for IP with port")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")

		hmac := auth.generateHMAC([]byte{}, "192.168.1.1")
		if hmac == "" {
			t.Error("HMAC should be generated for empty payload")
		}
	})

	t.Run("large payload", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")
		largePayload := make([]byte, 1024*1024) // 1MB
		for i := range largePayload {
			largePayload[i] = byte(i % 256)
		}

		hmac := auth.generateHMAC(largePayload, "192.168.1.1")
		if hmac == "" {
			t.Error("HMAC should be generated for large payload")
		}

		// HMAC should be hex-encoded SHA256 (64 characters)
		if len(hmac) != 64 {
			t.Errorf("HMAC length = %d, want 64", len(hmac))
		}
	})

	t.Run("consistency check", func(t *testing.T) {
		auth := NewHMACAuth("test-secret", "test-public")
		payload := []byte(`{"event":"click"}`)
		ip := "192.168.1.1"

		hmac1 := auth.generateHMAC(payload, ip)
		hmac2 := auth.generateHMAC(payload, ip)

		if hmac1 != hmac2 {
			t.Error("same payload and IP should produce same HMAC")
		}
	})
}
