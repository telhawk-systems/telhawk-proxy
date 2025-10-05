package sink

import (
	"context"

	"github.com/shortontech/gotrack/internal/event"
)

type Sink interface {
	Start(ctx context.Context) error
	Enqueue(e event.Event) error
	Close() error
	Name() string // Returns the sink name for metrics and logging
}
