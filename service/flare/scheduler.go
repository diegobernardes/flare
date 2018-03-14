package flare

import (
	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare/infra/config"
	base "github.com/diegobernardes/flare/scheduler"
	"github.com/diegobernardes/flare/scheduler/cluster"
	"github.com/diegobernardes/flare/scheduler/election"
)

type scheduler struct {
	cfg        *config.Client
	client     *base.Client
	locker     base.Locker
	cluster    base.Cluster
	dispatcher base.Dispatcher
	logger     log.Logger
}

func (s *scheduler) init() error {
	election := &election.Client{
		Eligible: s.cfg.GetBool("node.master.eligible"),
		Locker:   s.locker,
		Logger:   s.logger,
	}

	var err error
	election.Interval, err = s.cfg.GetDuration("node.master.election")
	if err != nil {
		panic(err)
	}

	election.KeepAlive, err = s.cfg.GetDuration("node.master.election-keep-alive")
	if err != nil {
		panic(err)
	}

	cluster := &cluster.Client{
		Cluster: s.cluster,
		Logger:  s.logger,
	}

	cluster.Interval, err = s.cfg.GetDuration("node.worker.register")
	if err != nil {
		panic(err)
	}

	cluster.KeepAlive, err = s.cfg.GetDuration("node.worker.register-keep-alive")
	if err != nil {
		panic(err)
	}

	cd := &base.ConsumerDispatcher{
		Cluster: s.cluster,
		Fetcher: s.dispatcher,
	}

	s.client = &base.Client{
		Election:           election,
		Cluster:            cluster,
		ConsumerDispatcher: cd,
	}
	if err := s.client.Init(); err != nil {
		panic(err)
	}

	s.client.Start()
	return nil
}

func (s *scheduler) initMaster() {

}

func (s *scheduler) stop() {
	s.client.Stop()
}
