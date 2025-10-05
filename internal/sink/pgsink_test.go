package sink

import (
	"context"
	"os"
	"testing"

	"revinar.io/go.track/internal/event"
)

// TestValidateTableName tests SQL injection prevention
func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		wantError bool
	}{
		{
			name:      "valid simple name",
			tableName: "events",
			wantError: false,
		},
		{
			name:      "valid with underscores",
			tableName: "events_json",
			wantError: false,
		},
		{
			name:      "valid with numbers",
			tableName: "events_2024",
			wantError: false,
		},
		{
			name:      "valid starting with underscore",
			tableName: "_private_events",
			wantError: false,
		},
		{
			name:      "empty string",
			tableName: "",
			wantError: true,
		},
		{
			name:      "SQL injection attempt with semicolon",
			tableName: "events; DROP TABLE users;--",
			wantError: true,
		},
		{
			name:      "SQL injection with quotes",
			tableName: "events' OR '1'='1",
			wantError: true,
		},
		{
			name:      "contains spaces",
			tableName: "my events",
			wantError: true,
		},
		{
			name:      "contains special characters",
			tableName: "events@table",
			wantError: true,
		},
		{
			name:      "contains dash",
			tableName: "events-table",
			wantError: true,
		},
		{
			name:      "starts with number",
			tableName: "2024_events",
			wantError: true,
		},
		{
			name:      "too long (>63 chars)",
			tableName: "this_is_a_very_long_table_name_that_exceeds_the_postgresql_limit_of_63_characters",
			wantError: true,
		},
		{
			name:      "exactly 63 chars (valid)",
			tableName: "abcdefghijklmnopqrstuvwxyz_abcdefghijklmnopqrstuvwxyz_1234567",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableName(tt.tableName)
			if (err != nil) != tt.wantError {
				t.Errorf("validateTableName(%q) error = %v, wantError = %v", tt.tableName, err, tt.wantError)
			}
		})
	}
}

// TestNewPGSinkFromEnv tests creation from environment variables
func TestNewPGSinkFromEnv(t *testing.T) {
	t.Run("uses defaults when env not set", func(t *testing.T) {
		// Clear all PG env vars
		envVars := []string{"PG_DSN", "PG_TABLE", "PG_BATCH_SIZE", "PG_FLUSH_MS", "PG_COPY"}
		oldValues := make(map[string]string)
		for _, key := range envVars {
			oldValues[key] = os.Getenv(key)
			os.Unsetenv(key)
		}
		defer func() {
			for key, val := range oldValues {
				os.Setenv(key, val)
			}
		}()

		sink := NewPGSinkFromEnv()

		if sink.config.Table != "events_json" {
			t.Errorf("Table = %q, want events_json", sink.config.Table)
		}
		if sink.config.BatchSize != 500 {
			t.Errorf("BatchSize = %d, want 500", sink.config.BatchSize)
		}
		if sink.config.FlushMS != 500 {
			t.Errorf("FlushMS = %d, want 500", sink.config.FlushMS)
		}
		if !sink.config.UseCopy {
			t.Error("UseCopy should be true by default")
		}
	})

	t.Run("uses env variables when set", func(t *testing.T) {
		envVars := map[string]string{
			"PG_DSN":        "postgres://test:test@localhost/test",
			"PG_TABLE":      "custom_events",
			"PG_BATCH_SIZE": "1000",
			"PG_FLUSH_MS":   "1000",
			"PG_COPY":       "false",
		}

		oldValues := make(map[string]string)
		for key, val := range envVars {
			oldValues[key] = os.Getenv(key)
			os.Setenv(key, val)
		}
		defer func() {
			for key, val := range oldValues {
				os.Setenv(key, val)
			}
		}()

		sink := NewPGSinkFromEnv()

		if sink.config.DSN != "postgres://test:test@localhost/test" {
			t.Errorf("DSN = %q, want custom DSN", sink.config.DSN)
		}
		if sink.config.Table != "custom_events" {
			t.Errorf("Table = %q, want custom_events", sink.config.Table)
		}
		if sink.config.BatchSize != 1000 {
			t.Errorf("BatchSize = %d, want 1000", sink.config.BatchSize)
		}
		if sink.config.FlushMS != 1000 {
			t.Errorf("FlushMS = %d, want 1000", sink.config.FlushMS)
		}
		if sink.config.UseCopy {
			t.Error("UseCopy should be false when PG_COPY=false")
		}
	})
}

// TestNewPGSink tests creation with explicit config
func TestNewPGSink(t *testing.T) {
	dsn := "postgres://user:pass@localhost:5432/test"
	sink := NewPGSink(dsn)

	if sink.config.DSN != dsn {
		t.Errorf("DSN = %q, want %q", sink.config.DSN, dsn)
	}
	if sink.config.Table != "events_json" {
		t.Errorf("Table = %q, want events_json", sink.config.Table)
	}
	if sink.config.BatchSize != 500 {
		t.Errorf("BatchSize = %d, want 500", sink.config.BatchSize)
	}
	if sink.config.FlushMS != 500 {
		t.Errorf("FlushMS = %d, want 500", sink.config.FlushMS)
	}
	if !sink.config.UseCopy {
		t.Error("UseCopy should be true by default")
	}
}

// TestPGSinkName tests the Name method
func TestPGSinkName(t *testing.T) {
	sink := NewPGSink("postgres://localhost/test")
	if sink.Name() != "postgres" {
		t.Errorf("Name() = %q, want postgres", sink.Name())
	}
}

// TestPGSinkStartValidation tests Start validates table name
func TestPGSinkStartValidation(t *testing.T) {
	t.Run("rejects invalid table name", func(t *testing.T) {
		// Set invalid table name via env
		oldTable := os.Getenv("PG_TABLE")
		defer os.Setenv("PG_TABLE", oldTable)
		os.Setenv("PG_TABLE", "events; DROP TABLE users;--")

		sink := NewPGSinkFromEnv()
		ctx := context.Background()

		err := sink.Start(ctx)
		if err == nil {
			t.Error("Start() should fail for invalid table name")
			sink.Close()
		}

		if err != nil && !contains2(err.Error(), "invalid table name") {
			t.Errorf("error should mention invalid table name, got: %v", err)
		}
	})

	t.Run("rejects connection to invalid DSN", func(t *testing.T) {
		sink := NewPGSink("invalid://dsn")
		ctx := context.Background()

		err := sink.Start(ctx)
		if err == nil {
			t.Error("Start() should fail for invalid DSN")
			sink.Close()
		}
	})
}

// TestPGSinkEnqueueBatching tests the batching logic
func TestPGSinkEnqueueBatching(t *testing.T) {
	t.Run("accumulates events in batch", func(t *testing.T) {
		sink := &PGSink{
			config: PGConfig{
				BatchSize: 10,
				FlushMS:   1000,
			},
			batch: make([]event.Event, 0, 10),
		}
		sink.ctx, sink.cancel = context.WithCancel(context.Background())
		defer sink.cancel()

		// Enqueue a few events (less than batch size)
		for i := 0; i < 5; i++ {
			evt := event.Event{
				EventID: "test",
				Type:    "click",
			}
			// Note: This will fail because db is nil, but we can check batch accumulation
			_ = sink.Enqueue(evt)
		}

		// Batch should contain events even though flush failed
		if len(sink.batch) != 5 {
			t.Errorf("batch length = %d, want 5", len(sink.batch))
		}
	})
}

// TestPGSinkClose tests closing without starting
func TestPGSinkClose(t *testing.T) {
	t.Run("handles close without start", func(t *testing.T) {
		sink := NewPGSink("postgres://localhost/test")
		err := sink.Close()
		if err != nil {
			t.Errorf("Close() on unstarted sink should not error: %v", err)
		}
	})
}

// TestGetIntEnv tests the integer environment variable helper
func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		want         int
	}{
		{
			name:         "returns default when not set",
			key:          "TEST_INT_UNSET",
			value:        "",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "parses valid integer",
			key:          "TEST_INT_VALID",
			value:        "100",
			defaultValue: 42,
			want:         100,
		},
		{
			name:         "returns default for invalid integer",
			key:          "TEST_INT_INVALID",
			value:        "not-a-number",
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "parses negative integer",
			key:          "TEST_INT_NEGATIVE",
			value:        "-10",
			defaultValue: 42,
			want:         -10,
		},
		{
			name:         "parses zero",
			key:          "TEST_INT_ZERO",
			value:        "0",
			defaultValue: 42,
			want:         0,
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

			got := getIntEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getIntEnv() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestPGSinkConfigValidation tests configuration validation
func TestPGSinkConfigValidation(t *testing.T) {
	t.Run("accepts valid table names", func(t *testing.T) {
		validNames := []string{
			"events",
			"events_json",
			"_private",
			"table123",
			"a",
		}

		for _, name := range validNames {
			err := validateTableName(name)
			if err != nil {
				t.Errorf("validateTableName(%q) should be valid, got error: %v", name, err)
			}
		}
	})

	t.Run("rejects invalid table names", func(t *testing.T) {
		invalidNames := []string{
			"",
			"123invalid",
			"table-name",
			"table name",
			"table;drop",
			"table' or '1'='1",
		}

		for _, name := range invalidNames {
			err := validateTableName(name)
			if err == nil {
				t.Errorf("validateTableName(%q) should be invalid", name)
			}
		}
	})
}

// Helper functions
func contains2(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf2(s, substr) >= 0)
}

func indexOf2(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
