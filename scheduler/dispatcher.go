package scheduler

import (
	"context"
	"os"
	"time"

	"github.com/kr/pretty"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/scheduler/node"
)

type DispatcherCluster interface {
	Nodes(ctx context.Context, time *time.Time) ([]node.Node, error)
}

type DispatcherStorager interface {
	Fetch(ctx context.Context, time *time.Time) ([]consumer.Consumer, error)
	Assign(ctx context.Context, consumerID, nodeID string) error
	// Unassign(ctx context.Context, consumerID string) error
}

type Dispatcher struct {
	Fetcher   DispatcherStorager
	Cluster   DispatcherCluster
	NodeID    string
	ctx       context.Context
	ctxCancel func()
}

func (cd *Dispatcher) Init() error {
	if cd.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if cd.Cluster == nil {
		return errors.New("missing Cluster")
	}

	if cd.Fetcher == nil {
		return errors.New("missing Fetcher")
	}

	return nil
}

func (cd *Dispatcher) Start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				go cd.Start()
			}
		}()

		var lastCheck *time.Time
		for {
			check := lastCheck
			t := time.Now()
			lastCheck = &t

			consumers, err := cd.Fetcher.Fetch(cd.ctx, check)
			if err != nil {
				panic(err)
			}

			nodes, err := cd.Cluster.Nodes(cd.ctx, check)
			if err != nil {
				panic(err)
			}

			if len(consumers) == 0 || len(nodes) == 0 {
				<-time.After(10 * time.Second)
				continue
			}

			rebalance := cd.genRebalance(consumers, nodes)
			if err := cd.execRebalance(rebalance); err != nil {
				pretty.Println(err.Error())
				os.Exit(1)
				panic(err)
			}

			pretty.Println(consumers, nodes)
			<-time.After(10 * time.Second)
		}
	}()
}

func (cd *Dispatcher) Stop() {
	// ir no banco e liberar geral que esta preso la.
	// talvez criar uma flag no cassandra como processamento, que temos que marcar false ou true.
	// dai o dispatcher verifica apenas esses caras, em algum momento eles vao sumir se pararem de
	// processar.
}

func (cd *Dispatcher) execRebalance(rebalance map[string][]string) error {
	for nodeID, consumersID := range rebalance {
		for _, consumerID := range consumersID {
			if err := cd.Fetcher.Assign(cd.ctx, nodeID, consumerID); err != nil {
				return errors.Wrapf(err, "error during assign consumer '%s' to node '%s'", consumerID, nodeID)
			}
		}
	}

	return nil
}

func (cd *Dispatcher) genRebalance(
	consumers []consumer.Consumer, nodes []node.Node,
) map[string][]string {
	result := make(map[string][]string)
	count := make(map[string]int)

	for _, node := range nodes {
		count[node.ID] = 0
	}

	for _, consumer := range consumers {
		if consumer.NodeID == "" {
			continue
		}
		count[consumer.NodeID] += 1
	}

	for _, consumer := range consumers {
		if cd.onCluster(consumer.NodeID, nodes) {
			continue
		}

		result[consumer.ID] = append(result[consumer.ID], cd.selectNode(count))
	}

	return result
}

func (cd *Dispatcher) onCluster(id string, nodes []node.Node) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func (cd *Dispatcher) selectNode(nodesCount map[string]int) string {
	var (
		key   string
		count int
	)

	for nodeKey, nodeCount := range nodesCount {
		if count <= nodeCount {
			key = nodeKey
			count = nodeCount
		}
	}

	return key
}
