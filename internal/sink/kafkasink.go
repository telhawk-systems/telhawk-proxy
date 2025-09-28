package sink

import (
	"context"
	"errors"

	"revinar.io/go.track/internal/event"
)

// KafkaSink is a placeholder Kafka producer. Wire brokers/topic later.
type KafkaSink struct {
	Brokers []string
	Topic   string
}

func NewKafkaSink(brokers []string, topic string) *KafkaSink {
	return &KafkaSink{Brokers: brokers, Topic: topic}
}

func (s *KafkaSink) Start(ctx context.Context) error {
	// TODO: create producer client
	return nil
}

func (s *KafkaSink) Enqueue(e event.Event) error {
	// TODO: serialize and send (key=event_id)
	return errors.New("kafka sink not implemented")
}

func (s *KafkaSink) Close() error {
	// TODO: flush and close producer
	return nil
}
