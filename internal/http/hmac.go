package httpx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// HMACAuth handles HMAC authentication for collection endpoints
type HMACAuth struct {
	secret       []byte
	publicKey    []byte
	requireHMAC  bool
}

// NewHMACAuth creates a new HMAC authentication handler
func NewHMACAuth(secret, publicKey string, requireHMAC bool) *HMACAuth {
	auth := &HMACAuth{
		secret:      []byte(secret),
		requireHMAC: requireHMAC,
	}
	
	// If public key is provided, decode it from base64
	if publicKey != "" {
		if decoded, err := base64.StdEncoding.DecodeString(publicKey); err == nil {
			auth.publicKey = decoded
		} else {
			log.Printf("WARNING: Invalid HMAC_PUBLIC_KEY format, using derived key")
		}
	}
	
	// If no public key provided or invalid, derive from secret
	if len(auth.publicKey) == 0 && len(auth.secret) > 0 {
		auth.publicKey = auth.derivePublicKey(auth.secret)
	}
	
	return auth
}

// derivePublicKey creates a public key from the secret using HKDF-like derivation
func (h *HMACAuth) derivePublicKey(secret []byte) []byte {
	// Use HMAC-SHA256 with a fixed salt to derive public key
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("gotrack-public-key-derivation"))
	return mac.Sum(nil)[:16] // Use first 16 bytes as public key
}

// GetPublicKeyBase64 returns the base64-encoded public key for client use
func (h *HMACAuth) GetPublicKeyBase64() string {
	if len(h.publicKey) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(h.publicKey)
}

// generateHMAC creates HMAC for payload using IP-derived key
func (h *HMACAuth) generateHMAC(payload []byte, clientIP string) string {
	if len(h.secret) == 0 {
		return ""
	}
	
	// Derive client-specific key from secret + IP
	derivedKey := h.deriveClientKey(clientIP)
	
	// Generate HMAC
	mac := hmac.New(sha256.New, derivedKey)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// deriveClientKey creates a client-specific key from secret + IP
func (h *HMACAuth) deriveClientKey(clientIP string) []byte {
	// Normalize IP (remove port, handle IPv6)
	ip := normalizeIP(clientIP)
	
	// Derive key: HMAC(secret, "client-key:" + ip)
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte("client-key:" + ip))
	return mac.Sum(nil)
}

// normalizeIP extracts and normalizes IP address
func normalizeIP(addr string) string {
	// Handle IPv6 with port: [::1]:8080 -> ::1
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]"); idx > 0 {
			return addr[1:idx]
		}
	}
	
	// Handle IPv4 with port: 192.168.1.1:8080 -> 192.168.1.1
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	
	// Return as-is if no port
	return addr
}

// VerifyHMAC validates the HMAC signature for a request
func (h *HMACAuth) VerifyHMAC(r *http.Request, payload []byte) bool {
	if !h.requireHMAC {
		return true // HMAC not required
	}
	
	if len(h.secret) == 0 {
		log.Printf("HMAC verification failed: no secret configured")
		return false
	}
	
	// Get HMAC from header
	providedHMAC := r.Header.Get("X-GoTrack-HMAC")
	if providedHMAC == "" {
		log.Printf("HMAC verification failed: missing X-GoTrack-HMAC header")
		return false
	}
	
	// Get client IP
	clientIP := getClientIP(r)
	
	// Generate expected HMAC
	expectedHMAC := h.generateHMAC(payload, clientIP)
	
	// Compare HMACs (constant time comparison)
	if !hmac.Equal([]byte(providedHMAC), []byte(expectedHMAC)) {
		log.Printf("HMAC verification failed for IP %s", clientIP)
		return false
	}
	
	log.Printf("HMAC verification successful for IP %s", clientIP)
	return true
}

// getClientIP extracts the real client IP considering proxies
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// GenerateClientScript generates JavaScript code for client-side HMAC generation
func (h *HMACAuth) GenerateClientScript() string {
	if len(h.publicKey) == 0 {
		return ""
	}
	
	publicKeyB64 := h.GetPublicKeyBase64()
	
	return fmt.Sprintf(`
// GoTrack HMAC Authentication
(function() {
  const GOTRACK_PUBLIC_KEY = '%s';
  
  // Simple HMAC-SHA256 implementation for client-side
  async function generateHMAC(payload, key) {
    const encoder = new TextEncoder();
    const keyData = encoder.encode(key);
    const payloadData = encoder.encode(payload);
    
    const cryptoKey = await crypto.subtle.importKey(
      'raw', keyData, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']
    );
    
    const signature = await crypto.subtle.sign('HMAC', cryptoKey, payloadData);
    return Array.from(new Uint8Array(signature))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }
  
  // Override fetch for GoTrack collection
  const originalFetch = window.fetch;
  window.fetch = async function(url, options = {}) {
    if (url.includes('/collect') && options.method === 'POST' && options.body) {
      try {
        const hmac = await generateHMAC(options.body, GOTRACK_PUBLIC_KEY);
        options.headers = options.headers || {};
        options.headers['X-GoTrack-HMAC'] = hmac;
      } catch (e) {
        console.warn('GoTrack HMAC generation failed:', e);
      }
    }
    return originalFetch.call(this, url, options);
  };
  
  console.log('GoTrack HMAC authentication initialized');
})();
`, publicKeyB64)
}