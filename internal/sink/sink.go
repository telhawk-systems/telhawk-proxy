package sink

import (
	"context"

	"revinar.io/go.track/internal/event"
)

type Sink interface {
	Start(ctx context.Context) error
	Enqueue(e event.Event) error
	Close() error
}
