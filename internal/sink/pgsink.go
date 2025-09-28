package sink

import (
	"context"
	"errors"

	"revinar.io/go.track/internal/event"
)

// PGSink is a placeholder Postgres sink. Wire DSN + batching later.
type PGSink struct {
	DSN string
}

func NewPGSink(dsn string) *PGSink { return &PGSink{DSN: dsn} }

func (s *PGSink) Start(ctx context.Context) error {
	// TODO: connect, prepare COPY/batch workers
	return nil
}

func (s *PGSink) Enqueue(e event.Event) error {
	// TODO: queue for batch insert / COPY
	return errors.New("pg sink not implemented")
}

func (s *PGSink) Close() error {
	// TODO: flush batches, close connections
	return nil
}
