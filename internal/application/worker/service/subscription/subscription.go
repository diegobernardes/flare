package subscription

import "context"

// Enqueuer enqueue the a message so it can be safely processed.
type Enqueuer interface {
	Enqueue(ctx context.Context, payload []byte) error
}
