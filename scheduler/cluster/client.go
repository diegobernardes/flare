package cluster

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type Client struct {
	Logger    log.Logger
	Interval  time.Duration
	KeepAlive time.Duration
	Cluster   cluster
	NodeID    string

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
}

func (m *Client) Init() error {
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())

	if m.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if m.Logger == nil {
		return errors.New("missing Logger")
	}
	m.Logger = log.With(m.Logger, "nodeID", m.NodeID)

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

func (m *Client) Start() {
	m.wg.Add(1)

	go func() {
		defer func() {
			m.wg.Done()
			if err := recover(); err != nil {
				go m.Start()
			}
		}()

		if err := m.Cluster.Join(m.ctx, m.NodeID, m.Interval); err != nil {
			level.Error(m.Logger).Log("message", "error during join node into cluster", "error", err.Error())
			<-time.After(m.Interval)
			go m.Start()
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

			if err := m.Cluster.KeepAlive(m.ctx, m.NodeID, m.Interval); err != nil {
				level.Error(m.Logger).Log("message", "error during node keep alive", "error", err.Error())
				continue
			}
			level.Debug(m.Logger).Log("message", "node cluster inscription refreshed")
		}
	}()
}

func (m *Client) Stop() {
	m.ctxCancel()
	m.wg.Wait()

	if err := m.Cluster.Leave(context.Background(), m.NodeID); err != nil {
		level.Error(m.Logger).Log("message", "error during leave cluster", "err", err.Error())
	}
}
