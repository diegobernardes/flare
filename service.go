package flare

import (
	baseLog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
)

type service struct {
	config   *config.Client
	logger   baseLog.Logger
	consumer *serviceConsumer
	external *external
}

func (s *service) init() error {
	s.consumer = &serviceConsumer{config: s.config, logger: s.logger, external: s.external}
	if err := s.consumer.init(); err != nil {
		return errors.Wrap(err, "error during consumer initialization")
	}
	return nil
}
