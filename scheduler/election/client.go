package election

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const electionLockKey = "node.master"

// Client is used to elect a master.
type Client struct {
	NodeID    string
	Eligible  bool
	Interval  time.Duration
	KeepAlive time.Duration
	Locker    locker

	isMaster  bool
	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
	mutex     sync.Mutex
	Logger    log.Logger
}

func (c *Client) Init() error {
	if c.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if c.Logger == nil {
		return errors.New("missing Logger")
	}
	c.Logger = log.With(c.Logger, "nodeID", c.NodeID)

	if c.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if c.KeepAlive <= 0 {
		return errors.New("invalid KeepAlive")
	}

	if c.Locker == nil {
		return errors.New("missing Locker")
	}

	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	return nil
}

func (c *Client) Start() {
	if !c.Eligible {
		return
	}
	go c.startElection()
	return
}

func (c *Client) Stop() {
	if !c.Eligible {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isMaster {
		if err := c.Locker.Release(c.ctx, electionLockKey, c.NodeID); err != nil {
			level.Error(c.Logger).Log(
				"message", "error during lock release", "err", err.Error(), "key", electionLockKey,
			)
		}
	}

	c.ctxCancel()
	c.wg.Wait()
}

func (c *Client) startElection() {
	c.wg.Add(1)

	defer func() {
		c.wg.Done()
		if err := recover(); err != nil {
			go c.startElection()
		}
	}()

	c.mutex.Lock()
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	c.mutex.Unlock()

	var (
		interval time.Duration
		next     bool
	)
	for {
		if c.isMaster {
			next, interval = c.electionAsMaster()
		} else {
			next, interval = c.electionToBeMaster()
		}
		if !next {
			break
		}

		select {
		case <-time.After(interval):
		case <-c.ctx.Done():
			break
		}
	}
}

func (c *Client) electionAsMaster() (bool, time.Duration) {
	locked, err := c.Locker.Refresh(c.ctx, electionLockKey, c.NodeID, c.Interval)
	if c.ctx.Err() != nil {
		return false, 0
	}
	if err != nil {
		level.Error(c.Logger).Log("message", "error during refresh lock", "error", err.Error())
	}

	if locked {
		level.Debug(c.Logger).Log("message", "master lock refreshed")
		return true, c.KeepAlive
	}

	level.Debug(c.Logger).Log("message", "master lost lock")
	c.isMaster = false
	return true, c.Interval
}

func (c *Client) electionToBeMaster() (bool, time.Duration) {
	locked, err := c.Locker.Lock(c.ctx, electionLockKey, c.NodeID, c.Interval)
	if c.ctx.Err() != nil {
		return false, 0
	}
	if err != nil {
		level.Error(c.Logger).Log("message", "error during acquire lock", "error", err.Error())
	}

	if locked {
		level.Debug(c.Logger).Log("message", "master lock acquired")
		c.isMaster = true
		return true, c.KeepAlive
	}

	level.Debug(c.Logger).Log("message", "someone has the master lock")
	return true, c.Interval
}
