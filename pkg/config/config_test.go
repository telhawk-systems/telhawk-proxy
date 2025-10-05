package config

import (
	"os"
	"testing"
)

func TestGetOr(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		defValue string
		want     string
	}{
		{
			name:     "returns env value when set",
			key:      "TEST_KEY_1",
			envValue: "from_env",
			defValue: "default",
			want:     "from_env",
		},
		{
			name:     "returns default when env not set",
			key:      "TEST_KEY_2_UNSET",
			envValue: "",
			defValue: "default",
			want:     "default",
		},
		{
			name:     "returns empty env value over default",
			key:      "TEST_KEY_3",
			envValue: "",
			defValue: "default",
			want:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			// Test
			got := getOr(tt.key, tt.defValue)
			if got != tt.want {
				t.Errorf("getOr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		defValue bool
		want     bool
	}{
		// True values
		{name: "recognizes '1' as true", key: "TEST_BOOL_1", envValue: "1", defValue: false, want: true},
		{name: "recognizes 't' as true", key: "TEST_BOOL_2", envValue: "t", defValue: false, want: true},
		{name: "recognizes 'true' as true", key: "TEST_BOOL_3", envValue: "true", defValue: false, want: true},
		{name: "recognizes 'y' as true", key: "TEST_BOOL_4", envValue: "y", defValue: false, want: true},
		{name: "recognizes 'yes' as true", key: "TEST_BOOL_5", envValue: "yes", defValue: false, want: true},
		{name: "recognizes 'TRUE' as true (case insensitive)", key: "TEST_BOOL_6", envValue: "TRUE", defValue: false, want: true},
		{name: "recognizes 'Yes' with spaces as true", key: "TEST_BOOL_7", envValue: " Yes ", defValue: false, want: true},

		// False values
		{name: "recognizes '0' as false", key: "TEST_BOOL_8", envValue: "0", defValue: true, want: false},
		{name: "recognizes 'f' as false", key: "TEST_BOOL_9", envValue: "f", defValue: true, want: false},
		{name: "recognizes 'false' as false", key: "TEST_BOOL_10", envValue: "false", defValue: true, want: false},
		{name: "recognizes 'n' as false", key: "TEST_BOOL_11", envValue: "n", defValue: true, want: false},
		{name: "recognizes 'no' as false", key: "TEST_BOOL_12", envValue: "no", defValue: true, want: false},
		{name: "recognizes 'FALSE' as false (case insensitive)", key: "TEST_BOOL_13", envValue: "FALSE", defValue: true, want: false},

		// Default values
		{name: "returns default when empty", key: "TEST_BOOL_14", envValue: "", defValue: true, want: true},
		{name: "returns default when unrecognized", key: "TEST_BOOL_15", envValue: "maybe", defValue: false, want: false},
		{name: "returns default when invalid", key: "TEST_BOOL_16", envValue: "xyz", defValue: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			// Test
			got := getBool(tt.key, tt.defValue)
			if got != tt.want {
				t.Errorf("getBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		defValue int64
		want     int64
	}{
		{
			name:     "parses valid positive integer",
			key:      "TEST_INT_1",
			envValue: "12345",
			defValue: 0,
			want:     12345,
		},
		{
			name:     "parses valid negative integer",
			key:      "TEST_INT_2",
			envValue: "-999",
			defValue: 0,
			want:     -999,
		},
		{
			name:     "parses zero",
			key:      "TEST_INT_3",
			envValue: "0",
			defValue: 100,
			want:     0,
		},
		{
			name:     "returns default when empty",
			key:      "TEST_INT_4",
			envValue: "",
			defValue: 42,
			want:     42,
		},
		{
			name:     "returns default when invalid",
			key:      "TEST_INT_5",
			envValue: "not_a_number",
			defValue: 99,
			want:     99,
		},
		{
			name:     "parses large number",
			key:      "TEST_INT_6",
			envValue: "9223372036854775807", // max int64
			defValue: 0,
			want:     9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			// Test
			got := getInt64(tt.key, tt.defValue)
			if got != tt.want {
				t.Errorf("getInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		defValue string
		want     []string
	}{
		{
			name:     "parses comma-separated values",
			key:      "TEST_SLICE_1",
			envValue: "log,kafka,postgres",
			defValue: "",
			want:     []string{"log", "kafka", "postgres"},
		},
		{
			name:     "trims whitespace",
			key:      "TEST_SLICE_2",
			envValue: " log , kafka , postgres ",
			defValue: "",
			want:     []string{"log", "kafka", "postgres"},
		},
		{
			name:     "returns single value",
			key:      "TEST_SLICE_3",
			envValue: "log",
			defValue: "",
			want:     []string{"log"},
		},
		{
			name:     "uses default when empty",
			key:      "TEST_SLICE_4",
			envValue: "",
			defValue: "default1,default2",
			want:     []string{"default1", "default2"},
		},
		{
			name:     "returns nil when both empty",
			key:      "TEST_SLICE_5",
			envValue: "",
			defValue: "",
			want:     nil,
		},
		{
			name:     "filters empty items",
			key:      "TEST_SLICE_6",
			envValue: "log,,kafka,  ,postgres",
			defValue: "",
			want:     []string{"log", "kafka", "postgres"},
		},
		{
			name:     "handles trailing comma",
			key:      "TEST_SLICE_7",
			envValue: "log,kafka,",
			defValue: "",
			want:     []string{"log", "kafka"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			// Test
			got := getStringSlice(tt.key, tt.defValue)

			// Compare slices
			if len(got) != len(tt.want) {
				t.Errorf("getStringSlice() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("getStringSlice()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Save current env
	oldEnv := make(map[string]string)
	envVars := []string{
		"SERVER_ADDR", "TRUST_PROXY", "DNT_RESPECT", "MAX_BODY_BYTES",
		"IP_HASH_SECRET", "OUTPUTS", "TEST_MODE", "ENABLE_HTTPS",
		"SSL_CERT_FILE", "SSL_KEY_FILE", "MIDDLEWARE_MODE",
		"FORWARD_DESTINATION", "AUTO_INJECT_PIXEL", "HMAC_SECRET",
		"REQUIRE_HMAC", "HMAC_PUBLIC_KEY", "METRICS_ENABLED",
		"METRICS_ADDR", "METRICS_TLS_CERT", "METRICS_TLS_KEY",
		"METRICS_CLIENT_CA", "METRICS_REQUIRE_TLS",
	}
	for _, key := range envVars {
		oldEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range oldEnv {
			if val != "" {
				os.Setenv(key, val)
			}
		}
	}()

	t.Run("loads defaults when no env vars set", func(t *testing.T) {
		cfg := Load()

		if cfg.ServerAddr != ":19890" {
			t.Errorf("ServerAddr = %v, want :19890", cfg.ServerAddr)
		}
		if cfg.TrustProxy != false {
			t.Errorf("TrustProxy = %v, want false", cfg.TrustProxy)
		}
		if cfg.DNTRespect != true {
			t.Errorf("DNTRespect = %v, want true", cfg.DNTRespect)
		}
		if cfg.MaxBodyBytes != 1<<20 {
			t.Errorf("MaxBodyBytes = %v, want %v", cfg.MaxBodyBytes, 1<<20)
		}
		if len(cfg.Outputs) != 1 || cfg.Outputs[0] != "log" {
			t.Errorf("Outputs = %v, want [log]", cfg.Outputs)
		}
	})

	t.Run("loads custom values from env", func(t *testing.T) {
		os.Setenv("SERVER_ADDR", ":8080")
		os.Setenv("TRUST_PROXY", "true")
		os.Setenv("DNT_RESPECT", "false")
		os.Setenv("MAX_BODY_BYTES", "2097152")
		os.Setenv("IP_HASH_SECRET", "my-secret")
		os.Setenv("OUTPUTS", "kafka,postgres")
		os.Setenv("TEST_MODE", "yes")
		os.Setenv("ENABLE_HTTPS", "1")
		os.Setenv("METRICS_ENABLED", "true")

		cfg := Load()

		if cfg.ServerAddr != ":8080" {
			t.Errorf("ServerAddr = %v, want :8080", cfg.ServerAddr)
		}
		if cfg.TrustProxy != true {
			t.Errorf("TrustProxy = %v, want true", cfg.TrustProxy)
		}
		if cfg.DNTRespect != false {
			t.Errorf("DNTRespect = %v, want false", cfg.DNTRespect)
		}
		if cfg.MaxBodyBytes != 2097152 {
			t.Errorf("MaxBodyBytes = %v, want 2097152", cfg.MaxBodyBytes)
		}
		if cfg.IPHashSecret != "my-secret" {
			t.Errorf("IPHashSecret = %v, want my-secret", cfg.IPHashSecret)
		}
		if len(cfg.Outputs) != 2 || cfg.Outputs[0] != "kafka" || cfg.Outputs[1] != "postgres" {
			t.Errorf("Outputs = %v, want [kafka postgres]", cfg.Outputs)
		}
		if cfg.TestMode != true {
			t.Errorf("TestMode = %v, want true", cfg.TestMode)
		}
		if cfg.EnableHTTPS != true {
			t.Errorf("EnableHTTPS = %v, want true", cfg.EnableHTTPS)
		}
		if cfg.MetricsEnabled != true {
			t.Errorf("MetricsEnabled = %v, want true", cfg.MetricsEnabled)
		}
	})
}
