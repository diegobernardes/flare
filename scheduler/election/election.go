package election

import (
	"context"
	"time"
)

type locker interface {
	Lock(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, nodeID string) error
}
