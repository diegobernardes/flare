package flare

import (
	"context"
	"runtime"
	"time"

	baseLog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/external/cassandra"
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

	config   *config.Client
	log      *log
	external *external
	infra    *infra
	logger   baseLog.Logger
	server   *server
	service  *service
}

// Init is used to initialize the Client.
func (c *Client) Init() error {
	nodeID := c.nodeID()

	c.config = &config.Client{Content: c.Config}
	if err := c.config.Init(); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}
	c.loadDefaultValues()

	c.log = &log{config: c.config}
	if err := c.log.init(); err != nil {
		return errors.Wrap(err, "error during log initialization")
	}
	c.logger = c.log.base

	c.external = &external{config: c.config}
	if err := c.external.init(); err != nil {
		return errors.Wrap(err, "error during external initialization")
	}

	c.infra = &infra{config: c.config, external: c.external, logger: c.logger, nodeID: nodeID}
	if err := c.infra.init(); err != nil {
		return errors.Wrap(err, "error during infra initialization")
	}

	c.service = &service{config: c.config, logger: c.logger, external: c.external}
	if err := c.service.init(); err != nil {
		return errors.Wrap(err, "error during service initialization")
	}

	c.server = &server{config: c.config, logger: c.logger}
	c.server.handler.consumer = c.service.consumer.apiClient
	if err := c.server.init(); err != nil {
		return errors.Wrap(err, "error during server initialization")
	}

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

	return nil
}

func (c *Client) Stop() error {
	c.infra.stop()

	// TODO: pegar o maior tempo de delete que roda async e colocar aqui para esperar.
	<-time.After(100 * time.Millisecond)
	c.external.stop()
	return nil
}

func (c *Client) Setup(ctx context.Context) error {
	cc := cassandra.Client{
		Hosts:         []string{"127.0.0.1"},
		Port:          9042,
		Keyspace:      "flare",
		AvoidKeyspace: true,
		Timeout:       time.Duration(1 * time.Minute),
	}

	if err := cc.Init(); err != nil {
		panic(err)
	}

	if err := cc.Start(); err != nil {
		panic(err)
	}

	if err := cc.Setup(); err != nil {
		panic(err)
	}

	return nil
}

func (c *Client) loadDefaultValues() {
	fn := func(key string, value interface{}) {
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

	fn("http.client.max-idle-connections", 1000)
	fn("http.client.max-idle-connections-per-host", 100)
	fn("http.client.idle-connection-timeout", "60s")

	fn("pagination.default-limit", 30)

	fn("producer.worker.spread.timeout", "10s")
	fn("producer.worker.spread.concurrency", 100)
	fn("producer.worker.spread.concurrency-output", 100)
	fn("producer.worker.delivery.timeout", "10s")
	fn("producer.worker.delivery.concurrency", 1000)

	fn("election.source", "cassandra")
	fn("election.eligible", true)
	fn("election.cassandra.ttl", "10s")
	fn("election.cassandra.renew", "5s")
	fn("election.cassandra.delete-timeout", "10s")

	fn("registry.source", "cassandra")
	fn("registry.cassandra.ttl", "10s")
	fn("registry.cassandra.renew", "5s")
	fn("registry.cassandra.delete-timeout", "10s")

	fn("repository.source", "cassandra")

	fn("queue.source", "sqs")
	fn("queue.sqs.producer.spread.name", "flare-producer-spread")
	fn("queue.sqs.producer.spread.ingress-timeout", "1s")
	fn("queue.sqs.producer.spread.egress.receive-wait-time", "20s")
	fn("queue.sqs.producer.delivery.name", "flare-producer-delivery")
	fn("queue.sqs.producer.delivery.ingress-timeout", "1s")
	fn("queue.sqs.producer.delivery.egress.receive-wait-time", "20s")

	fn("external.cassandra.hosts", []string{"127.0.0.1"})
	fn("external.cassandra.port", 9042)
	fn("external.cassandra.timeout", "600ms")
	fn("external.cassandra.keyspace", "flare")
}

func (c *Client) loadDefaultValuesSetup() {
	fn := func(key string, value interface{}) {
		if c.config.IsSet(key) {
			return
		}
		c.config.Set(key, value)
	}

	fn("provider.cassandra.avoidKeyspace", true)
	fn("external.cassandra.timeout", "10s")
}

func (c *Client) nodeID() string {
	return uuid.NewV4().String()
}
