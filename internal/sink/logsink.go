package sink

import (
	"context"
	"encoding/json"
	"log"

	"revinar.io/go.track/internal/event"
)

type LogSink struct{}

func NewLogSink() *LogSink { return &LogSink{} }

func (s *LogSink) Start(ctx context.Context) error { return nil }

func (s *LogSink) Enqueue(e event.Event) error {
	b, _ := json.Marshal(e)
	log.Printf("event %s", string(b))
	return nil
}

func (s *LogSink) Close() error { return nil }
