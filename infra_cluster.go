package flare

import (
	"context"
	"fmt"

	base "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/cluster"
	"github.com/diegobernardes/flare/infra/config"
)

type infraClusterExternal interface {
	consumerAssigner() cluster.ConsumerAssigner
	schedulerConsumerFetcher() cluster.SchedulerConsumerFetcher
	executeConsumerFetcher() cluster.ExecuteConsumerFetcher
	nodeFetcher() cluster.NodeFetcher
	guard(kind string) func(context.Context) context.Context
}

type infraCluster struct {
	config       *config.Client
	logger       base.Logger
	baseExternal *external
	nodeID       string

	external infraClusterExternal
	execute  cluster.Execute
	schedule cluster.Schedule
	registry cluster.LockTask
	election cluster.LockTask
}

func (ic *infraCluster) init() error {
	if ic.config == nil {
		return errors.New("missing config")
	}

	if ic.baseExternal == nil {
		return errors.New("missing baseExternal")
	}

	if ic.logger == nil {
		return errors.New("missing logger")
	}

	if ic.nodeID == "" {
		return errors.New("missing nodeID")
	}

	if err := ic.initExternal(); err != nil {
		return errors.Wrap(err, "error during external initialization")
	}

	if err := ic.initClusterTasks(); err != nil {
		return errors.Wrap(err, "error during tasks initialization")
	}

	return nil
}

func (ic *infraCluster) initExternal() error {
	source := ic.config.GetString("cluster.source")
	switch source {
	case externalEtcd:
		etcd := &infraClusterEtcd{
			config:   ic.config,
			external: ic.baseExternal,
			logger:   ic.logger,
			nodeID:   ic.nodeID,
		}

		if err := etcd.init(); err != nil {
			return errors.Wrap(err, "error during infraClusterEtcd initialization")
		}
		ic.external = etcd
		return nil
	default:
		return fmt.Errorf("invalid source '%s'", source)
	}
}

func (ic *infraCluster) initClusterTasks() error {
	if err := ic.initClusterRegistry(); err != nil {
		return errors.Wrap(err, "error during registry initialization")
	}

	if err := ic.initClusterElection(); err != nil {
		return errors.Wrap(err, "error during election initialization")
	}

	if err := ic.initClusterExecute(); err != nil {
		return errors.Wrap(err, "error during execute initialization")
	}

	if err := ic.initClusterSchedule(); err != nil {
		return errors.Wrap(err, "error during schedule initialization")
	}

	return nil
}

func (ic *infraCluster) initClusterExecute() error {
	ic.execute.ConsumerFetcher = ic.external.executeConsumerFetcher()
	ic.execute.Logger = base.With(ic.logger, "nodeID", ic.nodeID)
	ic.execute.NodeID = ic.nodeID

	if err := ic.execute.Init(); err != nil {
		return errors.Wrap(err, "error during cluster.Execute initialization")
	}
	return nil
}

func (ic *infraCluster) initClusterSchedule() error {
	ic.schedule.Node = ic.external.nodeFetcher()
	ic.schedule.Assigner = ic.external.consumerAssigner()
	ic.schedule.ConsumerFetcher = ic.external.schedulerConsumerFetcher()
	ic.schedule.Logger = base.With(ic.logger, "nodeID", ic.nodeID)

	if err := ic.schedule.Init(); err != nil {
		return errors.Wrap(err, "error during cluster.Schedule initialization")
	}
	return nil
}

func (ic *infraCluster) initClusterRegistry() error {
	ic.registry.Logger = base.With(ic.logger, "nodeID", ic.nodeID, "task", "registry")
	ic.registry.Guard = ic.external.guard("registry")
	ic.registry.Task = &cluster.GroupTask{
		Tasks: []cluster.Tasker{
			&ic.election,
			// &ic.execute,
		},
	}
	return nil
}

func (ic *infraCluster) initClusterElection() error {
	ic.election.Logger = base.With(ic.logger, "nodeID", ic.nodeID, "task", "election")
	ic.election.Guard = ic.external.guard("election")
	ic.election.Task = &cluster.GroupTask{
		Tasks: []cluster.Tasker{
			&ic.schedule,
		},
	}
	return nil
}

func (ic *infraCluster) start() {
	level.Info(ic.logger).Log("message", "starting infraCluster")
	ic.registry.Start()
}

func (ic *infraCluster) stop() {
	ic.registry.Stop()
}
