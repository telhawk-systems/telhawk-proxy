package sink

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

	"revinar.io/go.track/internal/event"
)

type LogSink struct {
	f   *os.File
	mu  sync.Mutex
	dst string
}

func NewLogSink() *LogSink {
	path := os.Getenv("LOG_PATH")
	if path == "" {
		path = "ndjson.log"
	} // default picked up from Docker env

	return &LogSink{dst: path}
}

func (s *LogSink) Start(ctx context.Context) error {
	if s.dst == "stdout" {
		return nil
	} // stdout only
	f, err := os.OpenFile(s.dst, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	s.f = f
	return nil
}

func (s *LogSink) Enqueue(e event.Event) error {
	b, _ := json.Marshal(e)
	line := append(b, '\n')
	if s.f != nil {
		s.mu.Lock()
		_, err := s.f.Write(line)
		s.mu.Unlock()
		return err
	}
	log.Printf("event %s", string(b))
	return nil
}

func (s *LogSink) Close() error {
	if s.f != nil {
		return s.f.Close()
	}
	return nil
}

func (s *LogSink) Name() string {
	return "log"
}
