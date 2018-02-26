package flare

import (
	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare/infra/config"
	base "github.com/diegobernardes/flare/scheduler"
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
	election := &base.Election{
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

	manager := &base.Manager{
		Cluster: s.cluster,
		Logger:  s.logger,
	}

	manager.Interval, err = s.cfg.GetDuration("node.worker.register")
	if err != nil {
		panic(err)
	}

	manager.KeepAlive, err = s.cfg.GetDuration("node.worker.register-keep-alive")
	if err != nil {
		panic(err)
	}

	cd := &base.ConsumerDispatcher{
		Cluster: s.cluster,
		Fetcher: s.dispatcher,
	}

	s.client = &base.Client{
		Election:           election,
		Manager:            manager,
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
