package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type ElectionStorager interface {
	Lock(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, nodeID string) error
}

const electionLockKey = "node.master"

// Election is used to elect a master.
type Election struct {
	NodeID    string
	Eligible  bool
	Interval  time.Duration
	KeepAlive time.Duration
	Locker    ElectionStorager
	Runner    Runner
	Logger    log.Logger

	isMaster      bool
	ctx           context.Context
	ctxCancel     func()
	wg            sync.WaitGroup
	mutex         sync.Mutex
	runnerStarted bool
	runnerMutex   sync.Mutex
}

func (e *Election) Init() error {
	if e.Runner == nil {
		return errors.New("missing Runner")
	}

	if e.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if e.Logger == nil {
		return errors.New("missing Logger")
	}
	e.Logger = log.With(e.Logger, "nodeID", e.NodeID)

	if e.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if e.KeepAlive <= 0 {
		return errors.New("invalid KeepAlive")
	}

	if e.Locker == nil {
		return errors.New("missing Locker")
	}

	e.ctx, e.ctxCancel = context.WithCancel(context.Background())
	return nil
}

func (e *Election) Start() {
	if !e.Eligible {
		return
	}
	go e.startElection()
	return
}

func (e *Election) Stop() {
	if !e.Eligible {
		return
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.isMaster {
		if err := e.Locker.Release(e.ctx, electionLockKey, e.NodeID); err != nil {
			level.Error(e.Logger).Log(
				"message", "error during lock release", "err", err.Error(), "key", electionLockKey,
			)
		}
	}

	e.ctxCancel()
	e.wg.Wait()
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
	locked, err := e.Locker.Refresh(e.ctx, electionLockKey, e.NodeID, e.Interval)
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
	e.runnerStop()
	e.isMaster = false
	return true, e.Interval
}

func (e *Election) electionToBeMaster() (bool, time.Duration) {
	locked, err := e.Locker.Lock(e.ctx, electionLockKey, e.NodeID, e.Interval)
	if e.ctx.Err() != nil {
		return false, 0
	}
	if err != nil {
		level.Error(e.Logger).Log("message", "error during acquire lock", "error", err.Error())
	}

	if locked {
		level.Debug(e.Logger).Log("message", "master lock acquired")
		e.runnerStart()
		e.isMaster = true
		return true, e.KeepAlive
	}

	level.Debug(e.Logger).Log("message", "someone has the master lock")
	return true, e.Interval
}

func (e *Election) runnerStart() {
	e.runnerMutex.Lock()
	defer e.runnerMutex.Unlock()

	if e.runnerStarted {
		return
	}
	e.Runner.Start()
	e.runnerStarted = true
}

func (e *Election) runnerStop() {
	e.runnerMutex.Lock()
	defer e.runnerMutex.Unlock()

	if !e.runnerStarted {
		return
	}
	e.Runner.Stop()
	e.runnerStarted = false
}
