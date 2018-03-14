package cluster

import (
	"context"
	"time"

	"github.com/diegobernardes/flare/scheduler/node"
)

type cluster interface {
	Join(ctx context.Context, id string, ttl time.Duration) error
	KeepAlive(ctx context.Context, id string, ttl time.Duration) error
	Leave(ctx context.Context, id string) error
	Nodes(ctx context.Context, time *time.Time) ([]node.Node, error)
}
