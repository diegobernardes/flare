package flare

import (
	"context"

	base "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/external/etcd"
	etcdCluster "github.com/diegobernardes/flare/external/etcd/infra/cluster"
	"github.com/diegobernardes/flare/infra/cluster"
	"github.com/diegobernardes/flare/infra/config"
)

type infraClusterEtcd struct {
	nodeID          string
	config          *config.Client
	external        *external
	logger          base.Logger
	consumer        *etcd.Consumer
	executeConsumer *etcdCluster.ExecuteConsumer
	node            *etcd.Node
	election        *etcd.Election
}

func (ice *infraClusterEtcd) init() error {
	if ice.nodeID == "" {
		return errors.New("missing nodeID")
	}

	if ice.config == nil {
		return errors.New("missing config")
	}

	if ice.external == nil {
		return errors.New("missing external")
	}

	if ice.logger == nil {
		return errors.New("missing logger")
	}

	registerTTL, err := ice.config.GetDuration("cluster.etcd.register-ttl")
	if err != nil {
		return errors.Wrap(err, "error during parse 'cluster.etcd.register-ttl'")
	}

	ice.node = &etcd.Node{
		ID:       ice.nodeID,
		Client:   ice.external.etcdClient,
		Logger:   ice.logger,
		LeaseTTL: registerTTL,
	}
	if err := ice.node.Init(); err != nil {
		return errors.Wrap(err, "error during etcd.Node initialization")
	}

	ice.consumer = &etcd.Consumer{
		Client: ice.external.etcdClient,
		Logger: ice.logger,
		Node:   ice.node,
	}
	if err := ice.node.Init(); err != nil {
		return errors.Wrap(err, "error during etcd.Consumer initialization")
	}

	ice.executeConsumer = &etcdCluster.ExecuteConsumer{Base: ice.consumer}
	if err := ice.executeConsumer.Init(); err != nil {
		return errors.Wrap(err, "error during execute consumer fetcher initialization")
	}

	ice.election = &etcd.Election{
		Client: ice.external.etcdClient,
		Logger: ice.logger,
		Node:   ice.node,
		NodeID: ice.nodeID,
		TTL:    10,
	}
	if err := ice.election.Init(); err != nil {
		return errors.Wrap(err, "error during etcd.Election initialization")
	}

	return nil
}

func (ice *infraClusterEtcd) consumerAssigner() cluster.ConsumerAssigner {
	return ice.consumer
}

func (ice *infraClusterEtcd) schedulerConsumerFetcher() cluster.SchedulerConsumerFetcher {
	return ice.consumer
}

func (ice *infraClusterEtcd) executeConsumerFetcher() cluster.ExecuteConsumerFetcher {
	return ice.executeConsumer
}

func (ice *infraClusterEtcd) nodeFetcher() cluster.NodeFetcher {
	return ice.node
}

func (ice *infraClusterEtcd) guard(kind string) func(context.Context) context.Context {
	switch kind {
	case "registry":
		return ice.node.Join
	case "election":
		return ice.election.Elect
	}
	return nil
}
