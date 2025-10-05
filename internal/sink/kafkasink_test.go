package sink

import (
	"os"
	"strings"
	"testing"
)

// TestNewKafkaSinkFromEnv tests creation from environment variables
func TestNewKafkaSinkFromEnv(t *testing.T) {
	t.Run("uses defaults when env not set", func(t *testing.T) {
		// Clear all Kafka env vars
		envVars := []string{
			"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_ACKS", "KAFKA_COMPRESSION",
			"KAFKA_SASL_MECHANISM", "KAFKA_SASL_USER", "KAFKA_SASL_PASSWORD",
			"KAFKA_TLS_CA", "KAFKA_TLS_SKIP_VERIFY",
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

		sink := NewKafkaSinkFromEnv()

		if len(sink.config.Brokers) != 1 || sink.config.Brokers[0] != "localhost:9092" {
			t.Errorf("Brokers = %v, want [localhost:9092]", sink.config.Brokers)
		}
		if sink.config.Topic != "gotrack.events" {
			t.Errorf("Topic = %q, want gotrack.events", sink.config.Topic)
		}
		if sink.config.Acks != "all" {
			t.Errorf("Acks = %q, want all", sink.config.Acks)
		}
	})

	t.Run("uses env variables when set", func(t *testing.T) {
		envVars := map[string]string{
			"KAFKA_BROKERS":         "broker1:9092,broker2:9092,broker3:9092",
			"KAFKA_TOPIC":           "custom.topic",
			"KAFKA_ACKS":            "1",
			"KAFKA_COMPRESSION":     "gzip",
			"KAFKA_SASL_MECHANISM":  "PLAIN",
			"KAFKA_SASL_USER":       "test-user",
			"KAFKA_SASL_PASSWORD":   "test-pass",
			"KAFKA_TLS_CA":          "/path/to/ca.pem",
			"KAFKA_TLS_SKIP_VERIFY": "true",
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

		sink := NewKafkaSinkFromEnv()

		expectedBrokers := []string{"broker1:9092", "broker2:9092", "broker3:9092"}
		if len(sink.config.Brokers) != 3 {
			t.Errorf("Brokers length = %d, want 3", len(sink.config.Brokers))
		}
		for i, broker := range expectedBrokers {
			if i >= len(sink.config.Brokers) || sink.config.Brokers[i] != broker {
				t.Errorf("Broker[%d] = %q, want %q", i, sink.config.Brokers[i], broker)
			}
		}

		if sink.config.Topic != "custom.topic" {
			t.Errorf("Topic = %q, want custom.topic", sink.config.Topic)
		}
		if sink.config.Acks != "1" {
			t.Errorf("Acks = %q, want 1", sink.config.Acks)
		}
		if sink.config.Compression != "gzip" {
			t.Errorf("Compression = %q, want gzip", sink.config.Compression)
		}
		if sink.config.SASLMechanism != "PLAIN" {
			t.Errorf("SASLMechanism = %q, want PLAIN", sink.config.SASLMechanism)
		}
		if sink.config.SASLUser != "test-user" {
			t.Errorf("SASLUser = %q, want test-user", sink.config.SASLUser)
		}
		if sink.config.SASLPassword != "test-pass" {
			t.Errorf("SASLPassword = %q, want test-pass", sink.config.SASLPassword)
		}
		if sink.config.TLSCAPath != "/path/to/ca.pem" {
			t.Errorf("TLSCAPath = %q, want /path/to/ca.pem", sink.config.TLSCAPath)
		}
		if !sink.config.TLSSkipVerify {
			t.Error("TLSSkipVerify should be true")
		}
	})

	t.Run("handles brokers with whitespace", func(t *testing.T) {
		oldBrokers := os.Getenv("KAFKA_BROKERS")
		defer func() {
			if oldBrokers != "" {
				os.Setenv("KAFKA_BROKERS", oldBrokers)
			} else {
				os.Unsetenv("KAFKA_BROKERS")
			}
		}()

		os.Setenv("KAFKA_BROKERS", "broker1:9092 , broker2:9092 ,  broker3:9092")

		sink := NewKafkaSinkFromEnv()

		expectedBrokers := []string{"broker1:9092", "broker2:9092", "broker3:9092"}
		if len(sink.config.Brokers) != 3 {
			t.Errorf("Brokers length = %d, want 3", len(sink.config.Brokers))
		}
		for i, broker := range expectedBrokers {
			if i >= len(sink.config.Brokers) || sink.config.Brokers[i] != broker {
				t.Errorf("Broker[%d] = %q, want %q", i, sink.config.Brokers[i], broker)
			}
		}
	})
}

// TestNewKafkaSink tests creation with explicit config
func TestNewKafkaSink(t *testing.T) {
	brokers := []string{"kafka1:9092", "kafka2:9092"}
	topic := "test.topic"

	sink := NewKafkaSink(brokers, topic)

	if len(sink.config.Brokers) != 2 {
		t.Errorf("Brokers length = %d, want 2", len(sink.config.Brokers))
	}
	if sink.config.Brokers[0] != "kafka1:9092" {
		t.Errorf("Brokers[0] = %q, want kafka1:9092", sink.config.Brokers[0])
	}
	if sink.config.Brokers[1] != "kafka2:9092" {
		t.Errorf("Brokers[1] = %q, want kafka2:9092", sink.config.Brokers[1])
	}
	if sink.config.Topic != "test.topic" {
		t.Errorf("Topic = %q, want test.topic", sink.config.Topic)
	}
	if sink.config.Acks != "all" {
		t.Errorf("Acks = %q, want all", sink.config.Acks)
	}
}

// TestKafkaSinkName tests the Name method
func TestKafkaSinkName(t *testing.T) {
	sink := NewKafkaSink([]string{"localhost:9092"}, "test")
	if sink.Name() != "kafka" {
		t.Errorf("Name() = %q, want kafka", sink.Name())
	}
}

// TestKafkaSinkClose tests closing without starting
func TestKafkaSinkClose(t *testing.T) {
	t.Run("handles close without start", func(t *testing.T) {
		sink := NewKafkaSink([]string{"localhost:9092"}, "test")
		err := sink.Close()
		if err != nil {
			t.Errorf("Close() on unstarted sink should not error: %v", err)
		}
	})
}

// TestGetEnvOr tests the string environment variable helper
func TestGetEnvOr(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "returns default when not set",
			key:          "TEST_STR_UNSET",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "returns env value when set",
			key:          "TEST_STR_SET",
			value:        "custom",
			defaultValue: "default",
			want:         "custom",
		},
		{
			name:         "returns empty string from env",
			key:          "TEST_STR_EMPTY",
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

			got := getEnvOr(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvOr() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetBoolEnv tests the boolean environment variable helper
func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		want         bool
	}{
		{
			name:         "returns default when not set",
			key:          "TEST_BOOL_UNSET",
			value:        "",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "recognizes '1' as true",
			key:          "TEST_BOOL_1",
			value:        "1",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes 't' as true",
			key:          "TEST_BOOL_T",
			value:        "t",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes 'true' as true",
			key:          "TEST_BOOL_TRUE",
			value:        "true",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes 'y' as true",
			key:          "TEST_BOOL_Y",
			value:        "y",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes 'yes' as true",
			key:          "TEST_BOOL_YES",
			value:        "yes",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes 'TRUE' as true (case insensitive)",
			key:          "TEST_BOOL_TRUE_UPPER",
			value:        "TRUE",
			defaultValue: false,
			want:         true,
		},
		{
			name:         "recognizes '0' as false",
			key:          "TEST_BOOL_0",
			value:        "0",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "recognizes 'f' as false",
			key:          "TEST_BOOL_F",
			value:        "f",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "recognizes 'false' as false",
			key:          "TEST_BOOL_FALSE",
			value:        "false",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "recognizes 'n' as false",
			key:          "TEST_BOOL_N",
			value:        "n",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "recognizes 'no' as false",
			key:          "TEST_BOOL_NO",
			value:        "no",
			defaultValue: true,
			want:         false,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_BOOL_INVALID",
			value:        "maybe",
			defaultValue: true,
			want:         true,
		},
		{
			name:         "handles whitespace",
			key:          "TEST_BOOL_WHITESPACE",
			value:        "  true  ",
			defaultValue: false,
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

			got := getBoolEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getBoolEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKafkaConfigMap tests configuration building
func TestKafkaConfigMap(t *testing.T) {
	t.Run("validates broker list parsing", func(t *testing.T) {
		oldBrokers := os.Getenv("KAFKA_BROKERS")
		defer func() {
			if oldBrokers != "" {
				os.Setenv("KAFKA_BROKERS", oldBrokers)
			} else {
				os.Unsetenv("KAFKA_BROKERS")
			}
		}()

		// Test single broker
		os.Setenv("KAFKA_BROKERS", "single:9092")
		sink := NewKafkaSinkFromEnv()
		if len(sink.config.Brokers) != 1 {
			t.Errorf("Single broker: got %d brokers, want 1", len(sink.config.Brokers))
		}

		// Test multiple brokers
		os.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
		sink = NewKafkaSinkFromEnv()
		if len(sink.config.Brokers) != 2 {
			t.Errorf("Multiple brokers: got %d brokers, want 2", len(sink.config.Brokers))
		}

		// Test brokers are properly joined with commas
		joined := strings.Join(sink.config.Brokers, ",")
		if joined != "broker1:9092,broker2:9092" {
			t.Errorf("Joined brokers = %q, want broker1:9092,broker2:9092", joined)
		}
	})

	t.Run("SASL configuration parsing", func(t *testing.T) {
		oldValues := make(map[string]string)
		saslVars := []string{"KAFKA_SASL_MECHANISM", "KAFKA_SASL_USER", "KAFKA_SASL_PASSWORD"}
		for _, key := range saslVars {
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

		// Test with SASL enabled
		os.Setenv("KAFKA_SASL_MECHANISM", "SCRAM-SHA-256")
		os.Setenv("KAFKA_SASL_USER", "test-user")
		os.Setenv("KAFKA_SASL_PASSWORD", "secret")

		sink := NewKafkaSinkFromEnv()

		if sink.config.SASLMechanism != "SCRAM-SHA-256" {
			t.Errorf("SASLMechanism = %q, want SCRAM-SHA-256", sink.config.SASLMechanism)
		}
		if sink.config.SASLUser != "test-user" {
			t.Errorf("SASLUser = %q, want test-user", sink.config.SASLUser)
		}
		if sink.config.SASLPassword != "secret" {
			t.Errorf("SASLPassword should be set but was %q", sink.config.SASLPassword)
		}
	})

	t.Run("TLS configuration parsing", func(t *testing.T) {
		oldValues := make(map[string]string)
		tlsVars := []string{"KAFKA_TLS_CA", "KAFKA_TLS_SKIP_VERIFY"}
		for _, key := range tlsVars {
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

		// Test with TLS enabled
		os.Setenv("KAFKA_TLS_CA", "/etc/kafka/ca.pem")
		os.Setenv("KAFKA_TLS_SKIP_VERIFY", "false")

		sink := NewKafkaSinkFromEnv()

		if sink.config.TLSCAPath != "/etc/kafka/ca.pem" {
			t.Errorf("TLSCAPath = %q, want /etc/kafka/ca.pem", sink.config.TLSCAPath)
		}
		if sink.config.TLSSkipVerify {
			t.Error("TLSSkipVerify should be false")
		}
	})
}
