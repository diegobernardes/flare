package cluster

import (
	"context"
	"errors"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Register is used to registry a node in the cluster. To leave, the context must be canceled or
// have a timeout. The context is used to indicate when the lease was lost for some unknow reason.
type Register interface {
	Join(ctx context.Context, id string) context.Context
}

// Registry is used to register the node on cluster.
type Registry struct {
	Register Register
	Logger   log.Logger
	NodeID   string
	Task     Tasker

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
}

// Init check the parameters to initialize the registry.
func (c *Registry) Init() error {
	if c.Register == nil {
		return errors.New("missing Register")
	}

	if c.Logger == nil {
		return errors.New("missing Logger")
	}

	if c.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if c.Task == nil {
		return errors.New("missing Runner")
	}

	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	return nil
}

// Start the registry.
func (c *Registry) Start() {
	c.wg.Add(1)

	defer func() {
		if err := recover(); err != nil {
			level.Error(c.Logger).Log("message", "panic during cluster registry", "reason", err)
			go c.Start()
		}
		c.wg.Done()
	}()

	var registred bool
	for {
		ctx, ctxCancel := context.WithCancel(context.Background())
		ctx = c.Register.Join(ctx, c.NodeID)
		if ctx.Err() == nil {
			c.Task.Start()
			registred = true
		}

		select {
		case <-ctx.Done():
			if registred {
				c.Task.Stop()
				registred = false
				level.Debug(c.Logger).Log("message", "lost node registration")
				continue
			}

			level.Error(c.Logger).Log("message", "error during registry", "error", ctx.Err())
		case <-c.ctx.Done():
			ctxCancel()
			return
		}
	}
}

// Stop the registry.
func (c *Registry) Stop() {
	c.Task.Stop()
	c.ctxCancel()
	c.wg.Wait()
}
