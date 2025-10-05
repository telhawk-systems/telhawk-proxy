package detection

import (
	"net/http"
	"strings"
)

// getClientIP extracts the client IP address from the request
// It considers proxy headers for more accurate IP detection
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common proxy header)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (original client)
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	// Check X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
