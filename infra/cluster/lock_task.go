package cluster

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// LockTask is a generic task that execute a group of tasks after acquire a lock.
type LockTask struct {
	// The returned context.Context should be used as a flag indicating that the lock is active.
	// If the parameter context.Context is cancelled or in case of any error inside the Guard, the
	// returned context.Context get canceled.
	Guard  func(context.Context) context.Context
	Logger log.Logger
	Task   Tasker

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
	isRunning bool
	mutex     sync.Mutex
}

// Init check if all the requirements are fulfilled.
func (lt *LockTask) Init() error {
	if lt.Guard == nil {
		return errors.New("missing Guard")
	}

	if lt.Logger == nil {
		return errors.New("missing Logger")
	}

	if lt.Task == nil {
		return errors.New("missing Task")
	}

	return nil
}

// Start the service.
func (lt *LockTask) Start() {
	lt.mutex.Lock()
	if lt.isRunning {
		lt.mutex.Unlock()
		return
	}
	lt.isRunning = true
	lt.mutex.Unlock()

	lt.ctx, lt.ctxCancel = context.WithCancel(context.Background())
	level.Info(lt.Logger).Log("message", "starting")
	lt.wg.Add(1)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(lt.Logger).Log(
					"message", "panic recovered", "error", err, "stacktrace", string(debug.Stack()),
				)

				if lt.ctx.Err() == nil {
					go lt.Start()
				}
			}
			lt.wg.Done()
		}()

		for {
			ctx := lt.Guard(lt.ctx)
			if err := ctx.Err(); err == nil {
				level.Info(lt.Logger).Log("message", "lock acquired")
				lt.Task.Start()
			} else {
				if lt.ctx.Err() == nil {
					level.Error(lt.Logger).Log("message", "error during lock initialization", "error", ctx.Err())
					continue
				}
				level.Debug(lt.Logger).Log("message", "stopping")
				return
			}

			<-ctx.Done()

			if lt.ctx.Err() == nil {
				level.Error(lt.Logger).Log("message", "lost lock", "error", ctx.Err())
				lt.Task.Stop()
			} else {
				level.Debug(lt.Logger).Log("message", "stopping")
				lt.Task.Stop()
				return
			}
		}
	}()
}

// Stop the service.
func (lt *LockTask) Stop() {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	if !lt.isRunning {
		return
	}

	lt.Task.Stop()
	lt.ctxCancel()
	lt.wg.Wait()
	lt.isRunning = false
}
