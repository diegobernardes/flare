package flare

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	base "github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer/api"
	"github.com/diegobernardes/flare/external/etcd"
	"github.com/diegobernardes/flare/infra/config"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/pagination"
)

type domain struct {
	config   *config.Client
	external *external
	logger   base.Logger
	nodeID   string

	consumerAPI *api.Client
}

func (d *domain) init() error {
	if d.config == nil {
		return errors.New("missing config")
	}

	if d.external == nil {
		return errors.New("missing external")
	}

	if d.logger == nil {
		return errors.New("missing logger")
	}

	if err := d.initConsumerAPI(); err != nil {
		return errors.Wrap(err, "error during consumerAPI initialization")
	}

	return nil
}

func (d *domain) initConsumerAPI() error {
	writer, err := infraHTTP.NewWriter(d.logger)
	if err != nil {
		return err
	}

	repository, err := d.initConsumerAPIClientRepository()
	if err != nil {
		return err
	}

	d.consumerAPI = &api.Client{
		Repository:      repository,
		GetID:           func(r *http.Request) string { return chi.URLParam(r, "id") },
		GetURI:          func(id string) string { return fmt.Sprintf("/consumers/%s", id) },
		ParsePagination: pagination.Parse(d.config.GetInt("pagination.default-limit")),
		Writer:          writer,
	}

	return d.consumerAPI.Init()
}

func (d *domain) initConsumerAPIClientRepository() (api.ClientRepository, error) {
	source := d.config.GetString("cluster.source")
	switch source {
	case externalEtcd:
		registerTTL, err := d.config.GetDuration("cluster.etcd.register-ttl")
		if err != nil {
			return nil, errors.Wrap(err, "error during parse 'cluster.etcd.register-ttl'")
		}

		node := &etcd.Node{
			Client:   d.external.etcdClient,
			Logger:   d.logger,
			ID:       d.nodeID,
			LeaseTTL: registerTTL,
		}

		if err := node.Init(); err != nil {
			return nil, errors.Wrap(err, "error during etcd.Node initialization")
		}

		consumer := &etcd.Consumer{
			Client: d.external.etcdClient,
			Logger: d.logger,
			Node:   node,
		}

		if err := consumer.Init(); err != nil {
			return nil, errors.Wrap(err, "error during etcd.Consumer initialization")
		}

		return consumer, nil
	default:
		return nil, fmt.Errorf("invalid source '%s'", source)
	}
}
