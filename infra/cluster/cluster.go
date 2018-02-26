package cluster

import "context"

// Tasker represent a work unit that can be started and stopped.
type Tasker interface {
	Start()
	Stop()
}

// Locker is used to lock a key for a given node. To release the lock, the context must be canceled
// or have a timeout. The context is used to indicate when the lock get released.
type Locker interface {
	Lock(ctx context.Context, key, nodeID string) context.Context
}
