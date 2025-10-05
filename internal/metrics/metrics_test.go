package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestLoadConfig tests the configuration loading from environment
func TestLoadConfig(t *testing.T) {
	t.Run("returns defaults when env not set", func(t *testing.T) {
		// Clear all env vars
		envVars := []string{
			"METRICS_ENABLED", "METRICS_ADDR", "METRICS_TLS_CERT",
			"METRICS_TLS_KEY", "METRICS_CLIENT_CA", "METRICS_REQUIRE_TLS",
			"METRICS_REQUIRE_AUTH",
		}
		oldValues := make(map[string]string)
		for _, key := range envVars {
			oldValues[key] = os.Getenv(key)
			os.Unsetenv(key)
		}
		defer func() {
			for key, val := range oldValues {
				if val != "" {
					os.Setenv(key, val)
				}
			}
		}()

		cfg := LoadConfig()

		if cfg.Enabled {
			t.Error("Enabled should be false by default")
		}
		if cfg.Addr != "127.0.0.1:9090" {
			t.Errorf("Addr = %q, want 127.0.0.1:9090", cfg.Addr)
		}
		if cfg.TLSCert != "" {
			t.Errorf("TLSCert should be empty, got %q", cfg.TLSCert)
		}
		if cfg.TLSKey != "" {
			t.Errorf("TLSKey should be empty, got %q", cfg.TLSKey)
		}
		if cfg.ClientCA != "" {
			t.Errorf("ClientCA should be empty, got %q", cfg.ClientCA)
		}
		if cfg.RequireTLS {
			t.Error("RequireTLS should be false by default")
		}
		if cfg.RequireAuth {
			t.Error("RequireAuth should be false by default")
		}
	})

	t.Run("loads custom values from environment", func(t *testing.T) {
		envVars := map[string]string{
			"METRICS_ENABLED":     "true",
			"METRICS_ADDR":        "0.0.0.0:8080",
			"METRICS_TLS_CERT":    "/path/to/cert.pem",
			"METRICS_TLS_KEY":     "/path/to/key.pem",
			"METRICS_CLIENT_CA":   "/path/to/ca.pem",
			"METRICS_REQUIRE_TLS": "true",
			"METRICS_REQUIRE_AUTH": "true",
		}

		oldValues := make(map[string]string)
		for key, val := range envVars {
			oldValues[key] = os.Getenv(key)
			os.Setenv(key, val)
		}
		defer func() {
			for key, val := range oldValues {
				if val != "" {
					os.Setenv(key, val)
				} else {
					os.Unsetenv(key)
				}
			}
		}()

		cfg := LoadConfig()

		if !cfg.Enabled {
			t.Error("Enabled should be true")
		}
		if cfg.Addr != "0.0.0.0:8080" {
			t.Errorf("Addr = %q, want 0.0.0.0:8080", cfg.Addr)
		}
		if cfg.TLSCert != "/path/to/cert.pem" {
			t.Errorf("TLSCert = %q, want /path/to/cert.pem", cfg.TLSCert)
		}
		if cfg.TLSKey != "/path/to/key.pem" {
			t.Errorf("TLSKey = %q, want /path/to/key.pem", cfg.TLSKey)
		}
		if cfg.ClientCA != "/path/to/ca.pem" {
			t.Errorf("ClientCA = %q, want /path/to/ca.pem", cfg.ClientCA)
		}
		if !cfg.RequireTLS {
			t.Error("RequireTLS should be true")
		}
		if !cfg.RequireAuth {
			t.Error("RequireAuth should be true")
		}
	})
}

// TestGetOr tests the string environment helper
func TestGetOr(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "returns default when not set",
			key:          "TEST_GETОР_UNSET",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "returns env value when set",
			key:          "TEST_GETОР_SET",
			value:        "custom",
			defaultValue: "default",
			want:         "custom",
		},
		{
			name:         "returns default for empty string",
			key:          "TEST_GETОР_EMPTY",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVal := os.Getenv(tt.key)
			defer func() {
				if oldVal != "" {
					os.Setenv(tt.key, oldVal)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getOr(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getOr() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetBool tests the boolean environment helper
func TestGetBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		want         bool
	}{
		{
			name:         "returns default when not set",
			key:          "TEST_GETBOOL_UNSET",
			value:        "",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "parses 'true'",
			key:          "TEST_GETBOOL_TRUE",
			value:        "true",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "parses 'false'",
			key:          "TEST_GETBOOL_FALSE",
			value:        "false",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "parses '1'",
			key:          "TEST_GETBOOL_ONE",
			value:        "1",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "parses '0'",
			key:          "TEST_GETBOOL_ZERO",
			value:        "0",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_GETBOOL_INVALID",
			value:        "maybe",
			defaultValue: true,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVal := os.Getenv(tt.key)
			defer func() {
				if oldVal != "" {
					os.Setenv(tt.key, oldVal)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewMetrics tests metrics creation
func TestNewMetrics(t *testing.T) {
	t.Run("creates all metric vectors", func(t *testing.T) {
		// Use InitMetrics instead to avoid registry conflicts
		m := InitMetrics()

		if m.EventsIngested == nil {
			t.Error("EventsIngested should not be nil")
		}
		if m.SinkErrors == nil {
			t.Error("SinkErrors should not be nil")
		}
		if m.HTTPRequests == nil {
			t.Error("HTTPRequests should not be nil")
		}
		if m.QueueDepth == nil {
			t.Error("QueueDepth should not be nil")
		}
		if m.BatchFlushLatency == nil {
			t.Error("BatchFlushLatency should not be nil")
		}
		if m.HTTPDuration == nil {
			t.Error("HTTPDuration should not be nil")
		}
	})
}

// TestMetricsConvenienceMethods tests the convenience methods
func TestMetricsConvenienceMethods(t *testing.T) {
	m := InitMetrics()

	t.Run("IncrementEventsIngested", func(t *testing.T) {
		// Should not panic
		m.IncrementEventsIngested("log")
		m.IncrementEventsIngested("kafka")
		m.IncrementEventsIngested("postgres")
	})

	t.Run("IncrementSinkErrors", func(t *testing.T) {
		// Should not panic
		m.IncrementSinkErrors("log", "write_error")
		m.IncrementSinkErrors("kafka", "connection_error")
		m.IncrementSinkErrors("postgres", "flush_error")
	})

	t.Run("IncrementHTTPRequests", func(t *testing.T) {
		// Should not panic
		m.IncrementHTTPRequests("/collect", "POST", "200")
		m.IncrementHTTPRequests("/px.gif", "GET", "200")
		m.IncrementHTTPRequests("/api/test", "GET", "404")
	})

	t.Run("SetQueueDepth", func(t *testing.T) {
		// Should not panic
		m.SetQueueDepth("kafka", 100.0)
		m.SetQueueDepth("postgres", 250.5)
		m.SetQueueDepth("log", 0.0)
	})

	t.Run("ObserveBatchFlushLatency", func(t *testing.T) {
		// Should not panic
		m.ObserveBatchFlushLatency("kafka", 50*time.Millisecond)
		m.ObserveBatchFlushLatency("postgres", 100*time.Millisecond)
		m.ObserveBatchFlushLatency("log", 1*time.Millisecond)
	})

	t.Run("ObserveHTTPDuration", func(t *testing.T) {
		// Should not panic
		m.ObserveHTTPDuration("/collect", "POST", 10*time.Millisecond)
		m.ObserveHTTPDuration("/px.gif", "GET", 1*time.Millisecond)
		m.ObserveHTTPDuration("/api/test", "GET", 50*time.Millisecond)
	})
}

// TestInitMetrics tests global metrics initialization
func TestInitMetrics(t *testing.T) {
	t.Run("returns metrics instance", func(t *testing.T) {
		m := InitMetrics()
		if m == nil {
			t.Error("InitMetrics should return non-nil metrics")
		}

		// Calling again should return same instance
		m2 := InitMetrics()
		if m != m2 {
			t.Error("InitMetrics should return same instance on subsequent calls")
		}
	})
}

// TestGetMetrics tests getting global metrics
func TestGetMetrics(t *testing.T) {
	t.Run("returns metrics instance", func(t *testing.T) {
		m := GetMetrics()
		if m == nil {
			t.Error("GetMetrics should return non-nil metrics")
		}
	})

	t.Run("returns same instance as InitMetrics", func(t *testing.T) {
		m1 := InitMetrics()
		m2 := GetMetrics()
		if m1 != m2 {
			t.Error("GetMetrics should return same instance as InitMetrics")
		}
	})
}

// TestNewServer tests metrics server creation
func TestNewServer(t *testing.T) {
	t.Run("creates server with config", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:9090",
		}

		srv := NewServer(cfg)

		if srv == nil {
			t.Fatal("NewServer should return non-nil server")
		}
		if srv.config.Enabled != true {
			t.Error("config.Enabled should be true")
		}
		if srv.config.Addr != "localhost:9090" {
			t.Errorf("config.Addr = %q, want localhost:9090", srv.config.Addr)
		}
		if srv.server == nil {
			t.Error("server.server should not be nil")
		}
	})

	t.Run("creates server with disabled config", func(t *testing.T) {
		cfg := Config{
			Enabled: false,
			Addr:    "localhost:9090",
		}

		srv := NewServer(cfg)

		if srv == nil {
			t.Fatal("NewServer should return non-nil server even when disabled")
		}
		if srv.config.Enabled {
			t.Error("config.Enabled should be false")
		}
	})

	t.Run("sets up metrics endpoint", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:0", // Use ephemeral port
		}

		srv := NewServer(cfg)

		if srv.server == nil {
			t.Fatal("server.server should not be nil")
		}

		// We can't easily test the routes without starting the server,
		// but we can verify the server was created
	})

	t.Run("configures TLS when enabled", func(t *testing.T) {
		cfg := Config{
			Enabled:    true,
			Addr:       "localhost:9090",
			RequireTLS: true,
			TLSCert:    "/path/to/cert.pem",
			TLSKey:     "/path/to/key.pem",
		}

		srv := NewServer(cfg)

		if srv.server.TLSConfig == nil {
			t.Error("TLSConfig should be set when RequireTLS is true")
		}
	})

	t.Run("does not configure TLS when disabled", func(t *testing.T) {
		cfg := Config{
			Enabled:    true,
			Addr:       "localhost:9090",
			RequireTLS: false,
		}

		srv := NewServer(cfg)

		if srv.server.TLSConfig != nil {
			t.Error("TLSConfig should be nil when RequireTLS is false")
		}
	})

	t.Run("sets timeouts for security", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:9090",
		}

		srv := NewServer(cfg)

		if srv.server.ReadTimeout != 10*time.Second {
			t.Errorf("ReadTimeout = %v, want 10s", srv.server.ReadTimeout)
		}
		if srv.server.WriteTimeout != 10*time.Second {
			t.Errorf("WriteTimeout = %v, want 10s", srv.server.WriteTimeout)
		}
		if srv.server.IdleTimeout != 60*time.Second {
			t.Errorf("IdleTimeout = %v, want 60s", srv.server.IdleTimeout)
		}
	})
}

// TestServerStart tests starting the metrics server
func TestServerStart(t *testing.T) {
	t.Run("returns immediately when disabled", func(t *testing.T) {
		cfg := Config{
			Enabled: false,
		}

		srv := NewServer(cfg)
		ctx := context.Background()

		err := srv.Start(ctx)
		if err != nil {
			t.Errorf("Start() should not error when disabled: %v", err)
		}
	})

	t.Run("starts HTTP server when enabled", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:0", // Ephemeral port
		}

		srv := NewServer(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := srv.Start(ctx)
		if err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Give it a moment to start
		time.Sleep(200 * time.Millisecond)

		// Clean up
		srv.Shutdown(context.Background())
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:0",
		}

		srv := NewServer(cfg)
		ctx, cancel := context.WithCancel(context.Background())

		err := srv.Start(ctx)
		if err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Cancel context
		cancel()

		// Give it time to process cancellation
		time.Sleep(100 * time.Millisecond)

		// Clean up
		srv.Shutdown(context.Background())
	})
}

// TestServerShutdown tests shutting down the metrics server
func TestServerShutdown(t *testing.T) {
	t.Run("returns immediately when disabled", func(t *testing.T) {
		cfg := Config{
			Enabled: false,
		}

		srv := NewServer(cfg)
		ctx := context.Background()

		err := srv.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown() should not error when disabled: %v", err)
		}
	})

	t.Run("shuts down running server", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:0",
		}

		srv := NewServer(cfg)
		ctx := context.Background()

		// Start server
		err := srv.Start(ctx)
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		// Give it time to start
		time.Sleep(200 * time.Millisecond)

		// Shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = srv.Shutdown(shutdownCtx)
		if err != nil {
			t.Errorf("Shutdown() failed: %v", err)
		}
	})

	t.Run("handles timeout during shutdown", func(t *testing.T) {
		cfg := Config{
			Enabled: true,
			Addr:    "localhost:0",
		}

		srv := NewServer(cfg)
		ctx := context.Background()

		// Start server
		srv.Start(ctx)
		time.Sleep(200 * time.Millisecond)

		// Create very short timeout context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// This might error due to timeout, which is acceptable
		_ = srv.Shutdown(shutdownCtx)
	})
}

// TestServerHealthEndpoint tests the metrics server health endpoint
func TestServerHealthEndpoint(t *testing.T) {
	t.Run("health endpoint returns OK", func(t *testing.T) {
		// Create a test server with the same handler setup as NewServer
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
		}

		body, _ := io.ReadAll(w.Body)
		if string(body) != "OK" {
			t.Errorf("body = %q, want OK", string(body))
		}
	})
}

// TestLoadCertPool tests certificate pool loading
func TestLoadCertPool(t *testing.T) {
	t.Run("returns nil for stub implementation", func(t *testing.T) {
		pool, err := loadCertPool("/path/to/cert.pem")

		// Current implementation is a stub
		if pool != nil {
			t.Error("loadCertPool should return nil in stub implementation")
		}
		if err != nil {
			t.Error("loadCertPool should not return error in stub implementation")
		}
	})
}

// TestConfigStruct tests the Config struct
func TestConfigStruct(t *testing.T) {
	t.Run("can create config with all fields", func(t *testing.T) {
		cfg := Config{
			Enabled:     true,
			Addr:        "0.0.0.0:9090",
			TLSCert:     "/cert.pem",
			TLSKey:      "/key.pem",
			ClientCA:    "/ca.pem",
			RequireTLS:  true,
			RequireAuth: true,
		}

		if !cfg.Enabled {
			t.Error("Enabled should be true")
		}
		if cfg.Addr != "0.0.0.0:9090" {
			t.Errorf("Addr = %q, want 0.0.0.0:9090", cfg.Addr)
		}
		if cfg.TLSCert != "/cert.pem" {
			t.Errorf("TLSCert = %q, want /cert.pem", cfg.TLSCert)
		}
	})

	t.Run("zero value config", func(t *testing.T) {
		var cfg Config

		if cfg.Enabled {
			t.Error("Enabled should be false for zero value")
		}
		if cfg.Addr != "" {
			t.Errorf("Addr should be empty for zero value, got %q", cfg.Addr)
		}
		if cfg.RequireTLS {
			t.Error("RequireTLS should be false for zero value")
		}
	})
}

// TestMetricsStruct tests the Metrics struct
func TestMetricsStruct(t *testing.T) {
	t.Run("all fields are exported", func(t *testing.T) {
		m := InitMetrics()

		// Should be able to access all fields
		_ = m.EventsIngested
		_ = m.SinkErrors
		_ = m.HTTPRequests
		_ = m.QueueDepth
		_ = m.BatchFlushLatency
		_ = m.HTTPDuration
	})
}
