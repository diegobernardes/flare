package flare

import (
	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare/infra/config"
	base "github.com/diegobernardes/flare/scheduler"
)

type scheduler struct {
	cfg        *config.Client
	client     *base.Dispatcher
	locker     base.Locker
	cluster    base.ClusterStorager
	dispatcher base.DispatcherStorager
	logger     log.Logger
}

func (s *scheduler) init() error {
	var (
		err    error
		nodeID = base.NodeID()
	)

	cluster := base.Cluster{
		Storage: s.cluster,
		Log:     s.logger,
		NodeID:  nodeID,
	}

	cluster.Interval, err = s.cfg.GetDuration("node.worker.register")
	if err != nil {
		panic(err)
	}

	cluster.KeepAlive, err = s.cfg.GetDuration("node.worker.register-keep-alive")
	if err != nil {
		panic(err)
	}

	election := &base.Election{
		Eligible: s.cfg.GetBool("node.master.eligible"),
		Locker:   s.locker,
		Logger:   s.logger,
		NodeID:   nodeID,
	}

	election.Interval, err = s.cfg.GetDuration("node.master.election")
	if err != nil {
		panic(err)
	}

	election.KeepAlive, err = s.cfg.GetDuration("node.master.election-keep-alive")
	if err != nil {
		panic(err)
	}

	proxy := &base.Proxy{
		Runners: []base.Runner{election},
	}
	if err := proxy.Init(); err != nil {
		panic(err)
	}
	cluster.Runner = proxy

	dispatcher := &base.Dispatcher{
		Cluster: s.cluster.(base.DispatcherCluster),
		NodeID:  nodeID,
		Fetcher: s.dispatcher,
	}
	election.Runner = dispatcher

	if err := dispatcher.Init(); err != nil {
		panic(err)
	}

	if err := election.Init(); err != nil {
		panic(err)
	}

	if err := cluster.Init(); err != nil {
		panic(err)
	}

	cluster.Start()
	return nil
}

func (s *scheduler) stop() {
	s.client.Stop()
}
