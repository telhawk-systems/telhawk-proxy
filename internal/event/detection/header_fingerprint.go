package detection

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
)

// generateHeaderFingerprint creates a fingerprint based on header names and values
func generateHeaderFingerprint(headers http.Header) string {
	// Create a fingerprint based on header names and order
	var headerParts []string
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Include only the header name and first few chars of value for fingerprinting
		value := headers.Get(key)
		if len(value) > 20 {
			value = value[:20] + "..."
		}
		headerParts = append(headerParts, key+":"+value)
	}

	fingerprint := strings.Join(headerParts, "|")
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:8]) // First 8 bytes as hex
}
