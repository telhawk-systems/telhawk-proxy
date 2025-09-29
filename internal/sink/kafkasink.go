package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"revinar.io/go.track/internal/event"
)

// KafkaConfig holds configuration for Kafka producer
type KafkaConfig struct {
	Brokers     []string
	Topic       string
	Acks        string
	Compression string
	
	// SASL config
	SASLMechanism string
	SASLUser      string
	SASLPassword  string
	
	// TLS config
	TLSCAPath      string
	TLSSkipVerify  bool
}

// KafkaSink produces events to Kafka with key=event_id for idempotency
type KafkaSink struct {
	config   KafkaConfig
	producer *kafka.Producer
}

// NewKafkaSinkFromEnv creates a KafkaSink from environment variables
func NewKafkaSinkFromEnv() *KafkaSink {
	brokersStr := os.Getenv("KAFKA_BROKERS")
	if brokersStr == "" {
		brokersStr = "localhost:9092"
	}
	brokers := strings.Split(brokersStr, ",")
	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}
	
	config := KafkaConfig{
		Brokers:        brokers,
		Topic:          getEnvOr("KAFKA_TOPIC", "gotrack.events"),
		Acks:           getEnvOr("KAFKA_ACKS", "all"),
		Compression:    getEnvOr("KAFKA_COMPRESSION", ""),
		SASLMechanism:  os.Getenv("KAFKA_SASL_MECHANISM"),
		SASLUser:       os.Getenv("KAFKA_SASL_USER"),
		SASLPassword:   os.Getenv("KAFKA_SASL_PASSWORD"),
		TLSCAPath:      os.Getenv("KAFKA_TLS_CA"),
		TLSSkipVerify:  getBoolEnv("KAFKA_TLS_SKIP_VERIFY", false),
	}
	
	return &KafkaSink{config: config}
}

// NewKafkaSink creates a KafkaSink with explicit configuration
func NewKafkaSink(brokers []string, topic string) *KafkaSink {
	return &KafkaSink{
		config: KafkaConfig{
			Brokers: brokers,
			Topic:   topic,
			Acks:    "all",
		},
	}
}

func (s *KafkaSink) Start(ctx context.Context) error {
	configMap := kafka.ConfigMap{
		"bootstrap.servers": strings.Join(s.config.Brokers, ","),
		"acks":             s.config.Acks,
		"retries":          10,
		"retry.backoff.ms": 100,
		"batch.size":       16384,
		"linger.ms":        10,
	}
	
	// Set compression if specified
	if s.config.Compression != "" {
		configMap["compression.type"] = s.config.Compression
	}
	
	// Configure SASL if specified
	if s.config.SASLMechanism != "" {
		configMap["security.protocol"] = "SASL_SSL"
		configMap["sasl.mechanism"] = s.config.SASLMechanism
		if s.config.SASLUser != "" {
			configMap["sasl.username"] = s.config.SASLUser
		}
		if s.config.SASLPassword != "" {
			configMap["sasl.password"] = s.config.SASLPassword
		}
	}
	
	// Configure TLS if CA path is provided
	if s.config.TLSCAPath != "" {
		if s.config.SASLMechanism == "" {
			configMap["security.protocol"] = "SSL"
		}
		configMap["ssl.ca.location"] = s.config.TLSCAPath
	}
	
	// Configure TLS verification
	if s.config.TLSSkipVerify {
		configMap["ssl.endpoint.identification.algorithm"] = "none"
	}
	
	producer, err := kafka.NewProducer(&configMap)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	
	s.producer = producer
	
	// Start delivery report handler in background
	go s.handleDeliveryReports(ctx)
	
	return nil
}

func (s *KafkaSink) Enqueue(e event.Event) error {
	if s.producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}
	
	// Serialize event to JSON
	value, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}
	
	// Create Kafka message with event_id as key for idempotency
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &s.config.Topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(e.EventID),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(e.Type)},
			{Key: "schema", Value: []byte("v1")},
		},
	}
	
	// Send message asynchronously
	err = s.producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}
	
	return nil
}

func (s *KafkaSink) Close() error {
	if s.producer == nil {
		return nil
	}
	
	// Flush any remaining messages (wait up to 10 seconds)
	remaining := s.producer.Flush(10 * 1000)
	if remaining > 0 {
		return fmt.Errorf("failed to flush %d remaining messages", remaining)
	}
	
	s.producer.Close()
	return nil
}

// handleDeliveryReports processes delivery reports in background
func (s *KafkaSink) handleDeliveryReports(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			e := s.producer.Events()
			select {
			case <-ctx.Done():
				return
			case ev := <-e:
				switch event := ev.(type) {
				case *kafka.Message:
					if event.TopicPartition.Error != nil {
						// In production, you might want to log this to a structured logger
						// or send to an error monitoring system
						fmt.Fprintf(os.Stderr, "Kafka delivery failed: %v\n", event.TopicPartition.Error)
					}
				case kafka.Error:
					// Kafka client errors
					fmt.Fprintf(os.Stderr, "Kafka error: %v\n", event)
				}
			}
		}
	}
}

// Helper functions
func getEnvOr(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "1", "t", "true", "y", "yes":
		return true
	case "0", "f", "false", "n", "no":
		return false
	}
	return defaultValue
}
