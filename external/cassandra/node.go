package cassandra

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/cluster"
)

type Node struct {
	Lease    *Lease
	Consumer *Consumer
	Logger   log.Logger
	Interval time.Duration
	state    []string
}

func (n *Node) Init() error {
	if n.Lease == nil {
		return errors.New("missing Lease")
	}

	if n.Consumer == nil {
		return errors.New("missing Consumer")
	}

	if n.Logger == nil {
		return errors.New("missing Logger")
	}

	if n.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	return nil
}

func (n *Node) Fetch(ctx context.Context, fn func(cluster.NodeStatus) error) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(n.Logger).Log("message", "panic during Fetch", "error", err)
				go n.Fetch(ctx, fn)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(n.Interval):
			}

			ids, err := n.Lease.nodes(ctx)
			if err != nil {
				level.Error(n.Logger).Log("message", "error during fetch nodes", "error", err)
				continue
			}

			for _, id := range ids {
				if member(id, n.state) {
					continue
				}

				if err = fn(cluster.NodeStatus{ID: id, Status: cluster.NodeStatusCreate}); err != nil {
					level.Error(n.Logger).Log("message", "error during process create", "error", err)
					continue
				}
				n.state = append(n.state, id)
			}

			for i := 0; i < len(n.state); i++ {
				id := n.state[i]
				if member(id, ids) {
					continue
				}

				if err = fn(cluster.NodeStatus{ID: id, Status: cluster.NodeStatusDelete}); err != nil {
					level.Error(n.Logger).Log("message", "error during process delete", "error", err)
					continue
				}

				n.state = append(n.state[:i], n.state[i+1:]...)
			}

			nodeIDS, err := n.Consumer.fetchNodeIDS(ctx)
			if err != nil {
				panic(err)
			}

			for i := 0; i < len(nodeIDS); i++ {
				nodeID := nodeIDS[i]

				if member(nodeID, n.state) {
					continue
				}

				if err := fn(cluster.NodeStatus{ID: nodeID, Status: cluster.NodeStatusDelete}); err != nil {
					level.Error(n.Logger).Log("message", "error during process delete", "error", err)
					continue
				}
			}
		}
	}()
}
