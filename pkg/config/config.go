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

func Load() Config {
	return Config{
		ServerAddr:   getOr("SERVER_ADDR", ":19890"),
		TrustProxy:   getBool("TRUST_PROXY", false),
		DNTRespect:   getBool("DNT_RESPECT", true),
		MaxBodyBytes: getInt64("MAX_BODY_BYTES", 1<<20), // 1 MiB default
		IPHashSecret: getOr("IP_HASH_SECRET", ""),       // set to enable hashing
	}
}
