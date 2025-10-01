package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerAddr   string
	TrustProxy   bool
	DNTRespect   bool
	MaxBodyBytes int64  // bytes for /collect payload
	IPHashSecret string // daily salt secret seed; if empty, we won’t hash
	Outputs      []string // enabled sinks: log, kafka, postgres
	TestMode     bool     // if true, generate test events on startup
	
	// HTTPS Configuration
	EnableHTTPS bool   // enable HTTPS server
	CertFile    string // path to SSL certificate file (server.crt)
	KeyFile     string // path to SSL private key file (server.key)
	
	// Middleware/Proxy Configuration
	MiddlewareMode     bool   // enable middleware mode - forward 404s to destination
	ForwardDestination string // destination hostname to forward non-tracking requests to
}

func getOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func getBool(k string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	switch v {
	case "1", "t", "true", "y", "yes":
		return true
	case "0", "f", "false", "n", "no":
		return false
	}
	return def
}
func getInt64(k string, def int64) int64 {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func getStringSlice(k, def string) []string {
	v := os.Getenv(k)
	if v == "" {
		v = def
	}
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func Load() Config {
	return Config{
		ServerAddr:   getOr("SERVER_ADDR", ":19890"),
		TrustProxy:   getBool("TRUST_PROXY", false),
		DNTRespect:   getBool("DNT_RESPECT", true),
		MaxBodyBytes: getInt64("MAX_BODY_BYTES", 1<<20), // 1 MiB default
		IPHashSecret: getOr("IP_HASH_SECRET", ""),       // set to enable hashing
		Outputs:      getStringSlice("OUTPUTS", "log"),  // default to log only
		TestMode:     getBool("TEST_MODE", false),       // enable test event generation
		
		// HTTPS Configuration
		EnableHTTPS: getBool("ENABLE_HTTPS", false),     // disabled by default
		CertFile:    getOr("SSL_CERT_FILE", "server.crt"), // default cert file path
		KeyFile:     getOr("SSL_KEY_FILE", "server.key"),   // default key file path
		
		// Middleware/Proxy Configuration
		MiddlewareMode:     getBool("MIDDLEWARE_MODE", false), // disabled by default
		ForwardDestination: getOr("FORWARD_DESTINATION", ""),  // no default destination
	}
}
