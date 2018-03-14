package scheduler

import (
	"context"
	"time"

	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/domain/consumer"
)

type Runner interface {
	Start()
	Stop()
}

type consumerProcessor interface {
	Process(consumer consumer.Consumer, payload []byte) error
}

// Locker is used to lock a given key within the cluster.
type Locker interface {
	Lock(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, nodeID string) error
}

func NodeID() string {
	return uuid.NewV4().String()
}
