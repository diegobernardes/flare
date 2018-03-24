package flare

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	baseLog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/infra/config"
)

// Variables set with ldflags during compilation.
var (
	Version   = ""
	BuildTime = ""
	Commit    = ""
	GoVersion = runtime.Version()
)

// Client is the entrypoint to start Flare.
type Client struct {
	Config string

	nodeID   string
	logger   baseLog.Logger
	config   *config.Client
	log      *log
	external *external
	infra    *infra
	domain   *domain
	server   *server
}

// Init is used to initialize the Client.
func (c *Client) Init() error {
	c.nodeID = uuid.NewV4().String()

	c.config = &config.Client{Content: c.Config}
	if err := c.config.Init(); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}
	if err := c.configValidateAndSetDefaultValues(); err != nil {
		return err
	}

	c.log = &log{config: c.config}
	if err := c.log.init(); err != nil {
		return errors.Wrap(err, "error during log initialization")
	}
	c.logger = c.log.base

	c.external = &external{config: c.config}
	if err := c.external.init(); err != nil {
		return errors.Wrap(err, "error during external initialization")
	}

	c.infra = &infra{config: c.config, external: c.external, logger: c.logger, nodeID: c.nodeID}
	if err := c.infra.init(); err != nil {
		return errors.Wrap(err, "error during infra initialization")
	}

	c.domain = &domain{config: c.config, external: c.external, logger: c.logger, nodeID: c.nodeID}
	if err := c.domain.init(); err != nil {
		return errors.Wrap(err, "error during domain initialization")
	}

	c.server = &server{config: c.config, logger: c.logger}
	c.server.handler.consumer = c.domain.consumerAPI
	if err := c.server.init(); err != nil {
		return errors.Wrap(err, "error during server initialization")
	}

	return nil
}

// Setup initialize the dependencies.
func (c *Client) Setup(ctx context.Context) error {
	return nil
}

// Start the service.
func (c *Client) Start() error {
	level.Info(c.logger).Log("message", "starting Flare")

	if err := c.external.start(); err != nil {
		return errors.Wrap(err, "error during external start")
	}

	c.infra.start()

	if err := c.server.start(); err != nil {
		return errors.Wrap(err, "error during server start")
	}

	level.Info(c.logger).Log("message", "Flare started")
	return nil
}

// Stop the service.
func (c *Client) Stop() error {
	return nil
}

func (c *Client) configValidateAndSetDefaultValues() error {
	var keys []string

	fn := func(key string, value interface{}) {
		keys = append(keys, key)
		if c.config.IsSet(key) {
			return
		}
		c.config.Set(key, value)
	}

	fn("log.level", "debug")
	fn("log.output", "stdout")
	fn("log.format", "human")

	fn("http.server.enable", true)
	fn("http.server.addr", ":8080")
	fn("http.server.timeout", "5s")

	fn("pagination.default-limit", 30)

	fn("consumer.state-source", "etcd")

	fn("producer.worker.enable", true)
	fn("producer.worker.spread.timeout", "10s")
	fn("producer.worker.spread.concurrency", 100)
	fn("producer.worker.spread.concurrency-output", 100)
	fn("producer.worker.delivery.timeout", "10s")
	fn("producer.worker.delivery.concurrency", 1000)

	fn("cluster.source", "etcd")
	fn("cluster.master-eligible", true)
	fn("cluster.etcd.register-ttl", "5s")
	fn("cluster.etcd.election-ttl", "10s")
	fn("cluster.etcd.election-ttl-refresh", "5s")

	fn("queue.source", "sqs")
	fn("queue.sqs.producer.spread.name", "flare-producer-spread")
	fn("queue.sqs.producer.spread.ingress-timeout", "5s")
	fn("queue.sqs.producer.spread.receive-wait-time", "20s")
	fn("queue.sqs.producer.delivery.name", "flare-producer-delivery")
	fn("queue.sqs.producer.delivery.ingress-timeout", "5s")
	fn("queue.sqs.producer.delivery.receive-wait-time", "20s")

	fn("external.aws.key", "")
	fn("external.aws.secret", "")
	fn("external.aws.region", "")

	fn("external.cassandra.hosts", []string{"127.0.0.1"})
	fn("external.cassandra.port", 9042)
	fn("external.cassandra.timeout", "600ms")
	fn("external.cassandra.keyspace", "flare")

	fn("external.etcd.addr", []string{"localhost:2379"})
	fn("external.etcd.dial-timeout", "5s")
	fn("external.etcd.username", "")
	fn("external.etcd.password", "")

	entries := c.config.UnknowEntries(keys)
	if len(entries) > 0 {
		return fmt.Errorf("invalid config entries '%s'", strings.Join(entries, "', '"))
	}
	return nil
}
