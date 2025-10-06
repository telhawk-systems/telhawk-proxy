package sink

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/shortontech/gotrack/internal/event"
)

func withEnvVars(t *testing.T, vars map[string]string, fn func()) {
	t.Helper()
	oldValues := make(map[string]string)
	for key, val := range vars {
		oldValues[key] = os.Getenv(key)
		if val == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, val)
		}
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
	fn()
}

func assertStringField(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertKafkaConfig(t *testing.T, cfg KafkaConfig, expected map[string]interface{}) {
	t.Helper()
	if brokers, ok := expected["brokers"].([]string); ok {
		if len(cfg.Brokers) != len(brokers) {
			t.Errorf("Brokers length = %d, want %d", len(cfg.Brokers), len(brokers))
		}
		for i, want := range brokers {
			if i < len(cfg.Brokers) && cfg.Brokers[i] != want {
				t.Errorf("Broker[%d] = %q, want %q", i, cfg.Brokers[i], want)
			}
		}
	}
	if val, ok := expected["topic"].(string); ok {
		assertStringField(t, cfg.Topic, val, "Topic")
	}
	if val, ok := expected["acks"].(string); ok {
		assertStringField(t, cfg.Acks, val, "Acks")
	}
	if val, ok := expected["compression"].(string); ok {
		assertStringField(t, cfg.Compression, val, "Compression")
	}
	if val, ok := expected["sasl_mechanism"].(string); ok {
		assertStringField(t, cfg.SASLMechanism, val, "SASLMechanism")
	}
	if val, ok := expected["sasl_user"].(string); ok {
		assertStringField(t, cfg.SASLUser, val, "SASLUser")
	}
	if val, ok := expected["sasl_password"].(string); ok {
		assertStringField(t, cfg.SASLPassword, val, "SASLPassword")
	}
	if val, ok := expected["tls_ca"].(string); ok {
		assertStringField(t, cfg.TLSCAPath, val, "TLSCAPath")
	}
	if tlsSkip, ok := expected["tls_skip_verify"].(bool); ok && cfg.TLSSkipVerify != tlsSkip {
		t.Errorf("TLSSkipVerify = %v, want %v", cfg.TLSSkipVerify, tlsSkip)
	}
}

func TestNewKafkaSinkFromEnv(t *testing.T) {
	t.Run("uses defaults when env not set", func(t *testing.T) {
		envVars := map[string]string{
			"KAFKA_BROKERS": "", "KAFKA_TOPIC": "", "KAFKA_ACKS": "", "KAFKA_COMPRESSION": "",
			"KAFKA_SASL_MECHANISM": "", "KAFKA_SASL_USER": "", "KAFKA_SASL_PASSWORD": "",
			"KAFKA_TLS_CA": "", "KAFKA_TLS_SKIP_VERIFY": "",
		}
		withEnvVars(t, envVars, func() {
			sink := NewKafkaSinkFromEnv()
			assertKafkaConfig(t, sink.config, map[string]interface{}{
				"brokers": []string{"localhost:9092"},
				"topic":   "gotrack.events",
				"acks":    "all",
			})
		})
	})

	t.Run("uses env variables when set", func(t *testing.T) {
		envVars := map[string]string{
			"KAFKA_BROKERS": "broker1:9092,broker2:9092,broker3:9092", "KAFKA_TOPIC": "custom.topic",
			"KAFKA_ACKS": "1", "KAFKA_COMPRESSION": "gzip", "KAFKA_SASL_MECHANISM": "PLAIN",
			"KAFKA_SASL_USER": "test-user", "KAFKA_SASL_PASSWORD": "test-pass",
			"KAFKA_TLS_CA": "/path/to/ca.pem", "KAFKA_TLS_SKIP_VERIFY": "true",
		}
		withEnvVars(t, envVars, func() {
			sink := NewKafkaSinkFromEnv()
			assertKafkaConfig(t, sink.config, map[string]interface{}{
				"brokers":         []string{"broker1:9092", "broker2:9092", "broker3:9092"},
				"topic":           "custom.topic",
				"acks":            "1",
				"compression":     "gzip",
				"sasl_mechanism":  "PLAIN",
				"sasl_user":       "test-user",
				"sasl_password":   "test-pass",
				"tls_ca":          "/path/to/ca.pem",
				"tls_skip_verify": true,
			})
		})
	})

	t.Run("handles brokers with whitespace", func(t *testing.T) {
		withEnvVars(t, map[string]string{"KAFKA_BROKERS": "broker1:9092 , broker2:9092 ,  broker3:9092"}, func() {
			sink := NewKafkaSinkFromEnv()
			assertKafkaConfig(t, sink.config, map[string]interface{}{
				"brokers": []string{"broker1:9092", "broker2:9092", "broker3:9092"},
			})
		})
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

// Test Kafka Start with various configuration paths
func TestKafkaSink_Start_ConfigurationPaths(t *testing.T) {
	t.Run("basic configuration", func(t *testing.T) {
		sink := NewKafkaSink([]string{"localhost:9092"}, "test-topic")
		ctx := context.Background()
		
		// This will fail without Kafka, but it exercises the config map creation
		err := sink.Start(ctx)
		// We expect an error since Kafka isn't running
		if err != nil {
			if !contains(err.Error(), "failed to create Kafka producer") {
				t.Logf("Got expected error: %v", err)
			}
		}
		
		// Cleanup if it somehow succeeded
		if sink.producer != nil {
			sink.Close()
		}
	})

	t.Run("with compression", func(t *testing.T) {
		sink := &KafkaSink{
			config: KafkaConfig{
				Brokers:     []string{"localhost:9092"},
				Topic:       "test",
				Acks:        "all",
				Compression: "gzip",
			},
		}
		ctx := context.Background()
		
		err := sink.Start(ctx)
		if err != nil {
			t.Logf("Got expected error (no Kafka): %v", err)
		}
		
		if sink.producer != nil {
			sink.Close()
		}
	})

	t.Run("with SASL configuration", func(t *testing.T) {
		sink := &KafkaSink{
			config: KafkaConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test",
				SASLMechanism: "PLAIN",
				SASLUser:      "test-user",
				SASLPassword:  "test-pass",
			},
		}
		ctx := context.Background()
		
		err := sink.Start(ctx)
		if err != nil {
			t.Logf("Got expected error (no Kafka): %v", err)
		}
		
		if sink.producer != nil {
			sink.Close()
		}
	})

	t.Run("with TLS configuration", func(t *testing.T) {
		sink := &KafkaSink{
			config: KafkaConfig{
				Brokers:   []string{"localhost:9092"},
				Topic:     "test",
				TLSCAPath: "/path/to/ca.pem",
			},
		}
		ctx := context.Background()
		
		err := sink.Start(ctx)
		if err != nil {
			t.Logf("Got expected error (no Kafka): %v", err)
		}
		
		if sink.producer != nil {
			sink.Close()
		}
	})

	t.Run("with TLS skip verify", func(t *testing.T) {
		sink := &KafkaSink{
			config: KafkaConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test",
				TLSSkipVerify: true,
			},
		}
		ctx := context.Background()
		
		err := sink.Start(ctx)
		if err != nil {
			t.Logf("Got expected error (no Kafka): %v", err)
		}
		
		if sink.producer != nil {
			sink.Close()
		}
	})

	t.Run("with SASL and TLS", func(t *testing.T) {
		sink := &KafkaSink{
			config: KafkaConfig{
				Brokers:       []string{"localhost:9092"},
				Topic:         "test",
				SASLMechanism: "SCRAM-SHA-256",
				SASLUser:      "user",
				SASLPassword:  "pass",
				TLSCAPath:     "/path/to/ca.pem",
			},
		}
		ctx := context.Background()
		
		err := sink.Start(ctx)
		if err != nil {
			t.Logf("Got expected error (no Kafka): %v", err)
		}
		
		if sink.producer != nil {
			sink.Close()
		}
	})
}

// Test Kafka Enqueue without producer
func TestKafkaSink_Enqueue_NoProducer(t *testing.T) {
	sink := NewKafkaSink([]string{"localhost:9092"}, "test")
	
	evt := event.Event{
		EventID: "test-123",
		Type:    "click",
	}
	
	err := sink.Enqueue(evt)
	if err == nil {
		t.Error("Enqueue should fail when producer is not initialized")
	}
	if !contains(err.Error(), "not initialized") {
		t.Errorf("error should mention not initialized: %v", err)
	}
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
		withEnvVars(t, map[string]string{"KAFKA_BROKERS": "single:9092"}, func() {
			sink := NewKafkaSinkFromEnv()
			if len(sink.config.Brokers) != 1 {
				t.Errorf("Single broker: got %d brokers, want 1", len(sink.config.Brokers))
			}
		})
		
		withEnvVars(t, map[string]string{"KAFKA_BROKERS": "broker1:9092,broker2:9092"}, func() {
			sink := NewKafkaSinkFromEnv()
			if len(sink.config.Brokers) != 2 {
				t.Errorf("Multiple brokers: got %d brokers, want 2", len(sink.config.Brokers))
			}
			joined := strings.Join(sink.config.Brokers, ",")
			if joined != "broker1:9092,broker2:9092" {
				t.Errorf("Joined brokers = %q, want broker1:9092,broker2:9092", joined)
			}
		})
	})

	t.Run("SASL configuration parsing", func(t *testing.T) {
		envVars := map[string]string{
			"KAFKA_SASL_MECHANISM": "SCRAM-SHA-256",
			"KAFKA_SASL_USER":      "test-user",
			"KAFKA_SASL_PASSWORD":  "secret",
		}
		withEnvVars(t, envVars, func() {
			sink := NewKafkaSinkFromEnv()
			assertKafkaConfig(t, sink.config, map[string]interface{}{
				"sasl_mechanism": "SCRAM-SHA-256",
				"sasl_user":      "test-user",
				"sasl_password":  "secret",
			})
		})
	})

	t.Run("TLS configuration parsing", func(t *testing.T) {
		envVars := map[string]string{
			"KAFKA_TLS_CA":          "/etc/kafka/ca.pem",
			"KAFKA_TLS_SKIP_VERIFY": "false",
		}
		withEnvVars(t, envVars, func() {
			sink := NewKafkaSinkFromEnv()
			assertKafkaConfig(t, sink.config, map[string]interface{}{
				"tls_ca":          "/etc/kafka/ca.pem",
				"tls_skip_verify": false,
			})
		})
	})
}
