package flare

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	baseLog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/external/cassandra"
	"github.com/diegobernardes/flare/infra/config"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/pagination"
	"github.com/diegobernardes/flare/service/consumer/api"
)

type serviceConsumer struct {
	logger    baseLog.Logger
	config    *config.Client
	apiClient *api.Client
	external  *external
}

func (sc *serviceConsumer) init() error {
	if err := sc.initAPI(); err != nil {
		return errors.Wrap(err, "error during api initialization")
	}
	return nil
}

func (sc *serviceConsumer) initAPI() error {
	writer, err := infraHTTP.NewWriter(sc.logger)
	if err != nil {
		return errors.Wrap(err, "error during http.Writer initialization")
	}

	repository, err := sc.initAPIRepository()
	if err != nil {
		return errors.Wrap(err, "error during repository initialize")
	}

	sc.apiClient = &api.Client{
		Writer:          writer,
		GetID:           func(r *http.Request) string { return chi.URLParam(r, "id") },
		GetURI:          func(id string) string { return fmt.Sprintf("/consumers/%s", id) },
		Repository:      repository,
		ParsePagination: pagination.Parse(sc.config.GetInt("pagination.default-limit")),
	}
	if err := sc.apiClient.Init(); err != nil {
		return errors.Wrap(err, "error during client initialization")
	}

	return nil
}

func (sc *serviceConsumer) initAPIRepository() (api.ClientRepositorier, error) {
	source := sc.config.GetString("repository.source")
	switch source {
	case externalCassandra:
		register, err := sc.initAPIRepositoryCassandra()
		if err != nil {
			return nil, errors.Wrap(err, "error during cassandra initialization")
		}
		return register, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}

func (sc *serviceConsumer) initAPIRepositoryCassandra() (api.ClientRepositorier, error) {
	repository := &cassandra.Consumer{
		Base:     sc.external.cassandraClient,
		Interval: 10 * time.Second,
	}
	if err := repository.Init(); err != nil {
		return nil, errors.Wrap(err, "error during repository initialization")
	}
	return repository, nil
}
