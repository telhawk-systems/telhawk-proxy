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
	}
}
