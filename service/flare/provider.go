package flare

import (
	"github.com/diegobernardes/flare/infra/config"
	"github.com/diegobernardes/flare/provider/cassandra"
	cassandraConsumer "github.com/diegobernardes/flare/provider/cassandra/domain/consumer"
	cassandraScheduler "github.com/diegobernardes/flare/provider/cassandra/scheduler"
	baseScheduler "github.com/diegobernardes/flare/scheduler"
)

type provider struct {
	cassandraClient                    *cassandra.Client
	cassandraDomainConsumerClient      *cassandraConsumer.Client
	cassandraSchedulerLockClient       *cassandraScheduler.Lock
	cassandraSchedulerNodeClient       *cassandraScheduler.Node
	cassandraSchedulerDispatcherClient *cassandraScheduler.Dispatcher
	cfg                                *config.Client
}

func (p *provider) cassandra() error {
	timeout, err := p.cfg.GetDuration("provider.cassandra.timeout")
	if err != nil {
		panic(err)
	}

	p.cassandraClient = &cassandra.Client{
		Hosts:    p.cfg.GetStringSlice("provider.cassandra.hosts"),
		Port:     p.cfg.GetInt("provider.cassandra.port"),
		Timeout:  timeout,
		Keyspace: p.cfg.GetString("provider.cassandra.keyspace"),
	}

	if err := p.cassandraClient.Init(); err != nil {
		panic(err)
	}

	p.cassandraDomainConsumerClient = &cassandraConsumer.Client{Base: p.cassandraClient}
	if err := p.cassandraDomainConsumerClient.Init(); err != nil {
		panic(err)
	}

	p.cassandraSchedulerLockClient = &cassandraScheduler.Lock{Client: p.cassandraClient}
	if err := p.cassandraSchedulerLockClient.Init(); err != nil {
		panic(err)
	}

	p.cassandraSchedulerNodeClient = &cassandraScheduler.Node{Client: p.cassandraClient}
	if err := p.cassandraSchedulerNodeClient.Init(); err != nil {
		panic(err)
	}

	p.cassandraSchedulerDispatcherClient = &cassandraScheduler.Dispatcher{Client: p.cassandraClient}
	if err := p.cassandraSchedulerDispatcherClient.Init(); err != nil {
		panic(err)
	}

	return nil
}

func (p *provider) getCassandraSchedulerLock() baseScheduler.Locker {
	if p.cassandraSchedulerLockClient != nil {
		return p.cassandraSchedulerLockClient
	}
	return nil
}

func (p *provider) getCassandraSchedulerCluster() baseScheduler.ClusterStorager {
	if p.cassandraSchedulerNodeClient != nil {
		return p.cassandraSchedulerNodeClient
	}
	return nil
}

func (p *provider) getCassandraSchedulerDispatcher() baseScheduler.DispatcherStorager {
	if p.cassandraSchedulerDispatcherClient != nil {
		return p.cassandraSchedulerDispatcherClient
	}
	return nil
}

func (p *provider) init() error {
	if err := p.cassandra(); err != nil {
		panic(err)
	}
	return nil
}
