// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type ClusterStorager interface {
	Join(ctx context.Context, id string, ttl time.Duration) error
	KeepAlive(ctx context.Context, id string, ttl time.Duration) error
	Leave(ctx context.Context, id string) error
}

// Cluster used to register the node at the cluster.
type Cluster struct {
	NodeID    string
	Log       log.Logger
	Interval  time.Duration
	KeepAlive time.Duration
	Storage   ClusterStorager
	Runner    Runner

	ctx           context.Context
	ctxCancel     func()
	wg            sync.WaitGroup
	runnerMutex   sync.Mutex
	runnerStarted bool
}

// Init is used to initialize the client.
func (m *Cluster) Init() error {
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())

	if m.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if m.Log == nil {
		return errors.New("missing Logger")
	}
	m.Log = log.With(m.Log, "nodeID", m.NodeID)

	if m.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if m.KeepAlive <= 0 {
		return errors.New("invalid KeepAlive")
	}

	if m.Storage == nil {
		return errors.New("missing Cluster")
	}

	if m.Runner == nil {
		return errors.New("missing Runner")
	}

	return nil
}

// Start is used to start the cluster registration.
func (m *Cluster) Start() {
	m.wg.Add(1)

	go func() {
		defer func() {
			m.wg.Done()
			m.runnerStop()

			if err := recover(); err != nil {
				level.Error(m.Log).Log("message", "catch panic at cluster processing", "error", err)
				go m.Start()
			}
		}()

		if err := m.Storage.Join(m.ctx, m.NodeID, m.Interval); err != nil {
			m.runnerStop()
			level.Error(m.Log).Log("message", "error during join node into cluster", "error", err.Error())
			<-time.After(m.Interval)
			go m.Start()
			return
		}
		level.Debug(m.Log).Log("message", "node joined cluster")
		m.runnerStart()

	loop:
		for {
			select {
			case <-m.ctx.Done():
				break loop
			case <-time.After(m.KeepAlive):
			}

			if err := m.Storage.KeepAlive(m.ctx, m.NodeID, m.Interval); err != nil {
				m.runnerStop()
				level.Error(m.Log).Log("message", "error during node keep alive", "error", err.Error())
				continue
			}
			level.Debug(m.Log).Log("message", "node cluster inscription refreshed")
			m.runnerStart()
		}
	}()
}

func (m *Cluster) runnerStart() {
	m.runnerMutex.Lock()
	defer m.runnerMutex.Unlock()

	if m.runnerStarted {
		return
	}
	m.Runner.Start()
	m.runnerStarted = true
}

func (m *Cluster) runnerStop() {
	m.runnerMutex.Lock()
	defer m.runnerMutex.Unlock()

	if !m.runnerStarted {
		return
	}
	m.Runner.Stop()
	m.runnerStarted = false
}

// Stop is used to stop the cluster registration.
func (m *Cluster) Stop() {
	m.ctxCancel()
	m.wg.Wait()
	m.Runner.Stop()

	if err := m.Storage.Leave(context.Background(), m.NodeID); err != nil {
		level.Error(m.Log).Log("message", "error during leave cluster", "err", err.Error())
	}
}
