package flare

import (
	"fmt"
	"time"

	base "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/external/cassandra"
	"github.com/diegobernardes/flare/infra/cluster"
	"github.com/diegobernardes/flare/infra/config"
)

type infraCluster struct {
	config   *config.Client
	logger   base.Logger
	external *external
	nodeID   string
	registry *cluster.Registry
}

func (ic *infraCluster) init() error {
	register, err := ic.initRegister()
	if err != nil {
		return errors.Wrap(err, "error during cluster.Register initialization")
	}

	locker, err := ic.initLocker()
	if err != nil {
		return errors.Wrap(err, "error during cluster.Locker initialization")
	}

	consumerFetcher, err := ic.initConsumerFetcher()
	if err != nil {
		return errors.Wrap(err, "error during cluster.ConsumerFetcher initialization")
	}

	nodeFetcher, err := ic.initNodeFetcher()
	if err != nil {
		return errors.Wrap(err, "error during cluster.NodeFetcher initialization")
	}

	schedule := &cluster.Schedule{
		Logger:   ic.logger,
		Consumer: consumerFetcher,
		Node:     nodeFetcher,
	}

	if err := schedule.Init(); err != nil {
		return errors.Wrap(err, "error during cluster.Schedule initialization")
	}

	election := &cluster.Election{
		Logger: ic.logger,
		NodeID: ic.nodeID,
		Locker: locker,
		Task: &cluster.Task{
			Tasks: []cluster.Tasker{schedule},
		},
	}

	if err := election.Init(); err != nil {
		return errors.Wrap(err, "error during cluster.Election initialization")
	}

	ic.registry = &cluster.Registry{
		Logger:   ic.logger,
		NodeID:   ic.nodeID,
		Register: register,
		Task: &cluster.Task{
			Tasks: []cluster.Tasker{election},
		},
	}
	if err := ic.registry.Init(); err != nil {
		return errors.Wrap(err, "error during cluster.Registry initialization")
	}

	return nil
}

func (ic *infraCluster) initRegister() (cluster.Register, error) {
	source := ic.config.GetString("registry.source")
	switch source {
	case externalCassandra:
		register, err := ic.initCassandraLease("registry")
		if err != nil {
			return nil, errors.Wrap(err, "error during cassandra initialization")
		}
		return register, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}

func (ic *infraCluster) initLocker() (cluster.Locker, error) {
	source := ic.config.GetString("election.source")
	switch source {
	case externalCassandra:
		locker, err := ic.initCassandraLease("election")
		if err != nil {
			return nil, errors.Wrap(err, "error during cassandra initialization")
		}
		return locker, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}

func (ic *infraCluster) initConsumerFetcher() (cluster.ConsumerFetcher, error) {
	source := ic.config.GetString("repository.source")
	switch source {
	case externalCassandra:
		consumerFetcher, err := ic.initConsumerFetcherCassandra()
		if err != nil {
			return nil, errors.Wrap(err, "error during cassandra initialization")
		}
		return consumerFetcher, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}

func (ic *infraCluster) initNodeFetcher() (cluster.NodeFetcher, error) {
	source := ic.config.GetString("repository.source")
	switch source {
	case externalCassandra:
		nodeFetcher, err := ic.initNodeFetcherCassandra()
		if err != nil {
			return nil, errors.Wrap(err, "error during cassandra initialization")
		}
		return nodeFetcher, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}

func (ic *infraCluster) initNodeFetcherCassandra() (*cassandra.Node, error) {
	lease, err := ic.initCassandraLease("registry")
	if err != nil {
		return nil, errors.Wrap(err, "error during cassandra initialization")
	}

	consumer, err := ic.initConsumerFetcherCassandra()
	if err != nil {
		return nil, errors.Wrap(err, "error during cassandra initialization")
	}

	node := &cassandra.Node{
		Consumer: consumer,
		Lease:    lease,
		Interval: 5 * time.Second,
		Logger:   ic.logger,
	}

	if err := node.Init(); err != nil {
		return nil, errors.Wrap(err, "error during node fetcher initialization")
	}
	return node, nil
}

func (ic *infraCluster) initConsumerFetcherCassandra() (*cassandra.Consumer, error) {
	consumer := &cassandra.Consumer{
		Base:     ic.external.cassandraClient,
		Interval: 5 * time.Second,
		Logger:   ic.logger,
	}

	if err := consumer.Init(); err != nil {
		return nil, errors.Wrap(err, "error during consumer fetcher initialization")
	}
	return consumer, nil
}

func (ic *infraCluster) initCassandraLease(kind string) (*cassandra.Lease, error) {
	lease := &cassandra.Lease{
		Client: ic.external.cassandraClient,
		Logger: ic.logger,
	}

	var err error
	lease.DeleteTimeout, err = ic.config.GetDuration(fmt.Sprintf("%s.cassandra.delete-timeout", kind))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid '%s.cassandra.delete-timeout'", kind)
	}

	lease.Renew, err = ic.config.GetDuration(fmt.Sprintf("%s.cassandra.renew", kind))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid '%s.cassandra.renew'", kind)
	}

	lease.TTL, err = ic.config.GetDuration(fmt.Sprintf("%s.cassandra.ttl", kind))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid '%s.cassandra.ttl'", kind)
	}

	if err := lease.Init(); err != nil {
		return nil, errors.Wrapf(err, "error during cassandra %s initialization", kind)
	}
	return lease, nil
}

func (ic *infraCluster) start() {
	go ic.registry.Start()
}

func (ic *infraCluster) stop() {
	ic.registry.Stop()
}
