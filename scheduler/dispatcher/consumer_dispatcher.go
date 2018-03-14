package dispatcher

import (
	"context"
	"os"
	"time"

	"github.com/kr/pretty"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/scheduler/node"
)

/*
	vai varrer todos os consumers de tempos em tempos para pegar todos que pertencem ao nó.
	tentar lockar eles e iniciar o processamento.
	se lockar, processar.

	se qnd atualizar, nao existir mais na bse local, liberar o lock e parar de processar.
*/

/*
	no inicio pegamos todos os consumers e os nós.
	se o consumer estiver associado a um nó que nao existe, vamos associar ele para outro nó.
	se um nó sair do cluster, temos que tirar todos os consumers (bater a transaction no creatd at).
*/

type ConsumerDispatcher struct {
	Fetcher   Dispatcher
	Cluster   Cluster
	nodeID    string
	ctx       context.Context
	ctxCancel func()
}

func (cd *ConsumerDispatcher) start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				go cd.start()
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

func (cd *ConsumerDispatcher) Stop() {

}

func (cd *ConsumerDispatcher) init() error {
	if cd.Cluster == nil {
		return errors.New("missing Cluster")
	}

	if cd.Fetcher == nil {
		return errors.New("missing Fetcher")
	}

	return nil
}

func (cd *ConsumerDispatcher) execRebalance(rebalance map[string][]string) error {
	for nodeID, consumersID := range rebalance {
		for _, consumerID := range consumersID {
			if err := cd.Fetcher.Assign(cd.ctx, nodeID, consumerID); err != nil {
				return errors.Wrapf(err, "error during assign consumer '%s' to node '%s'", consumerID, nodeID)
			}
		}
	}

	return nil
}

func (cd *ConsumerDispatcher) genRebalance(
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

func (cd *ConsumerDispatcher) onCluster(id string, nodes []node.Node) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func (cd *ConsumerDispatcher) selectNode(nodesCount map[string]int) string {
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
