package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const electionLockKey = "node.master"

// Election is used to election a master at the cluster.
type Election struct {
	nodeID    string
	isMaster  bool
	Eligible  bool
	Interval  time.Duration
	KeepAlive time.Duration
	Locker    Locker

	ctx       context.Context
	ctxCancel func()

	wg     sync.WaitGroup
	mutex  sync.Mutex
	Logger log.Logger
}

func (e *Election) startElection() {
	e.wg.Add(1)

	defer func() {
		e.wg.Done()
		if err := recover(); err != nil {
			go e.startElection()
		}
	}()

	e.mutex.Lock()
	e.ctx, e.ctxCancel = context.WithCancel(context.Background())
	e.mutex.Unlock()

	var (
		interval time.Duration
		next     bool
	)
	for {
		if e.isMaster {
			next, interval = e.electionAsMaster()
		} else {
			next, interval = e.electionToBeMaster()
		}
		if !next {
			break
		}

		select {
		case <-time.After(interval):
		case <-e.ctx.Done():
			break
		}
	}
}

func (e *Election) electionAsMaster() (bool, time.Duration) {
	locked, err := e.Locker.Refresh(e.ctx, electionLockKey, e.nodeID, e.Interval)
	if e.ctx.Err() != nil {
		return false, 0
	}
	if err != nil {
		level.Error(e.Logger).Log("message", "error during refresh lock", "error", err.Error())
	}

	if locked {
		level.Debug(e.Logger).Log("message", "master lock refreshed")
		return true, e.KeepAlive
	}

	level.Debug(e.Logger).Log("message", "master lost lock")
	e.isMaster = false
	return true, e.Interval
}

func (e *Election) electionToBeMaster() (bool, time.Duration) {
	locked, err := e.Locker.Lock(e.ctx, electionLockKey, e.nodeID, e.Interval)
	if e.ctx.Err() != nil {
		return false, 0
	}
	if err != nil {
		level.Error(e.Logger).Log("message", "error during acquire lock", "error", err.Error())
	}

	if locked {
		level.Debug(e.Logger).Log("message", "master lock acquired")
		e.isMaster = true
		return true, e.KeepAlive
	}

	level.Debug(e.Logger).Log("message", "someone has the master lock")
	return true, e.Interval
}

func (e *Election) start() {
	if !e.Eligible {
		return
	}
	go e.startElection()
	return
}

func (e *Election) stop() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if !e.Eligible {
		return
	}

	if e.isMaster {
		if err := e.Locker.Release(e.ctx, electionLockKey, e.nodeID); err != nil {
			level.Error(e.Logger).Log(
				"message", "error during lock release", "err", err.Error(), "key", electionLockKey,
			)
		}
	}

	e.ctxCancel()
	e.wg.Wait()
}

func (e *Election) init() error {
	if e.Logger == nil {
		return errors.New("missing Logger")
	}
	e.Logger = log.With(e.Logger, "nodeID", e.nodeID)

	if e.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if e.KeepAlive <= 0 {
		return errors.New("invalid KeepAlive")
	}

	if e.Locker == nil {
		return errors.New("missing Locker")
	}

	return nil
}
