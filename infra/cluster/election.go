package cluster

import (
	"context"
	"errors"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Election is used to elect a master on cluster.
type Election struct {
	Locker Locker
	NodeID string
	Logger log.Logger
	Task   Tasker

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
}

// Init check the parameters to initialize the election.
func (e *Election) Init() error {
	if e.Locker == nil {
		return errors.New("missing Locker")
	}

	if e.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if e.Task == nil {
		return errors.New("missing Task")
	}

	e.ctx, e.ctxCancel = context.WithCancel(context.Background())
	return nil
}

// Start the election.
func (e *Election) Start() {
	e.wg.Add(1)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(e.Logger).Log("message", "panic during master election", "reason", err)
				e.Start()
			}
			e.wg.Done()
		}()

		var locked bool
		for {
			ctx, ctxCancel := context.WithCancel(context.Background())
			ctx = e.Locker.Lock(ctx, "/node/master", e.NodeID)
			if ctx.Err() == nil {
				locked = true
				e.Task.Start()
			}

			select {
			case <-ctx.Done():
				if locked {
					locked = false
					e.Task.Stop()
					level.Debug(e.Logger).Log("message", "lost master")
					continue
				}

				level.Error(e.Logger).Log("message", "error during master election", "error", ctx.Err())
			case <-e.ctx.Done():
				ctxCancel()
				return
			}
		}
	}()
}

// Stop the election.
func (e *Election) Stop() {
	e.ctxCancel()
	e.wg.Wait()
}
