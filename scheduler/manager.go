package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

// esse manager tem que virar algo como node manager, algo que remta o nó.

type Manager struct {
	Logger    log.Logger
	Interval  time.Duration
	KeepAlive time.Duration
	Cluster   Cluster
	nodeID    string
	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
}

func (m *Manager) start() {
	m.wg.Add(1)

	go func() {
		defer func() {
			m.wg.Done()
			if err := recover(); err != nil {
				go m.start()
			}
		}()

		if err := m.Cluster.Join(m.ctx, m.nodeID, m.Interval); err != nil {
			level.Error(m.Logger).Log("message", "error during join node into cluster", "error", err.Error())
			<-time.After(m.Interval)
			go m.start()
			return
		}
		level.Debug(m.Logger).Log("message", "node joined cluster")

	loop:
		for {
			select {
			case <-m.ctx.Done():
				break loop
			case <-time.After(m.KeepAlive):
			}

			if err := m.Cluster.KeepAlive(m.ctx, m.nodeID, m.Interval); err != nil {
				level.Error(m.Logger).Log("message", "error during node keep alive", "error", err.Error())
				continue
			}
			level.Debug(m.Logger).Log("message", "node cluster inscription refreshed")
		}
	}()
}

func (m *Manager) stop() {
	m.ctxCancel()
	m.wg.Wait()
}

func (m *Manager) init() error {
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())

	if m.Logger == nil {
		return errors.New("missing Logger")
	}
	m.Logger = log.With(m.Logger, "nodeID", m.nodeID)

	if m.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if m.KeepAlive <= 0 {
		return errors.New("invalid KeepAlive")
	}

	if m.Cluster == nil {
		return errors.New("missing Cluster")
	}

	return nil
}
