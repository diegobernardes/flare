package lease

import (
	"context"
)

type Mutexer interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}

// implementar no provider etcd o lock
