package flare

import (
	base "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
)

type infra struct {
	config       *config.Client
	external     *external
	infraCluster *infraCluster
	logger       base.Logger
	nodeID       string
}

func (i *infra) init() error {
	i.infraCluster = &infraCluster{
		config:       i.config,
		logger:       i.logger,
		baseExternal: i.external,
		nodeID:       i.nodeID,
	}

	if err := i.infraCluster.init(); err != nil {
		return errors.Wrap(err, "error during infra initialization")
	}

	return nil
}

func (i *infra) start() {
	level.Info(i.logger).Log("message", "starting infra")
	i.infraCluster.start()
}

func (i *infra) stop() {
	i.infraCluster.stop()
}
