package flare

import (
	"time"

	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare/infra/config"
	base "github.com/diegobernardes/flare/scheduler"
)

type scheduler struct {
	cfg              *config.Client
	client           *base.DispatcherMaster
	locker           base.Locker
	cluster          base.ClusterStorager
	dispatcher       base.DispatcherMasterStorager
	dispatcherWorker base.DispatcherWorkerStorager
	logger           log.Logger
	clusterScheduler base.Cluster
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

	worker := &base.DispatcherWorker{
		NodeID:   nodeID,
		Interval: 10 * time.Second,
		Storager: s.dispatcherWorker,
	}

	if err := worker.Init(); err != nil {
		panic(err)
	}

	proxy := &base.Proxy{
		Runners: []base.Runner{election, worker},
	}
	if err := proxy.Init(); err != nil {
		panic(err)
	}
	cluster.Runner = proxy

	dispatcher := &base.DispatcherMaster{
		Cluster: s.cluster.(base.DispatcherMasterCluster),
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
	s.clusterScheduler = cluster
	return nil
}

func (s *scheduler) stop() {
	s.clusterScheduler.Stop()
}
