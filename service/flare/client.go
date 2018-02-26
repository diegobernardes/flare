package flare

import (
	"context"
	"runtime"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
	"github.com/diegobernardes/flare/provider/cassandra"
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
	config     *config.Client
	domain     *domain
	provider   *provider
	scheduler  *scheduler
	logger     log.Logger
	loggerInfo log.Logger
}

// Start the service.
func (c *Client) Start() error {
	c.loggerInfo.Log("message", "starting service")

	c.provider.cfg = c.config
	if err := c.provider.init(); err != nil {
		panic(err)
	}

	c.domain.provider = c.provider
	c.domain.logger = c.logger
	if err := c.domain.init(); err != nil {
		panic(err)
	}

	c.scheduler.logger = c.logger
	c.scheduler.cfg = c.config
	c.scheduler.locker = c.provider.getCassandraSchedulerLock()
	c.scheduler.cluster = c.provider.getCassandraSchedulerCluster()
	c.scheduler.dispatcher = c.provider.getCassandraSchedulerDispatcher()
	if err := c.scheduler.init(); err != nil {
		panic(err)
	}

	s := server{}
	s.handler.consumer = c.domain.consumer
	s.logger = c.logger
	s.cfg = c.config
	if err := s.init(); err != nil {
		panic(err)
	}

	return nil
}

// Stop the service.
func (c *Client) Stop() error {
	c.scheduler.stop()
	return nil
}

// Setup is used to bootstrap the service.
func (c *Client) Setup(ctx context.Context) error {
	cass := &cassandra.Client{
		Hosts: []string{"127.0.0.1"},
		Port:  9042,
		// Timeout:       1000 * time.Millisecond,
		Keyspace:      "flare",
		AvoidKeyspace: true,
	}

	if err := cass.Init(); err != nil {
		panic(err)
	}

	if err := cass.Setup(); err != nil {
		panic(err)
	}

	return nil
}

// Init is used to initialize the Client.
func (c *Client) Init(cfg string) error {
	c.provider = &provider{}
	c.domain = &domain{}
	c.scheduler = &scheduler{}
	c.config = &config.Client{Content: cfg}

	if err := c.config.Init(); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}
	c.loadDefaultValues()

	if err := c.initLogger(); err != nil {
		return errors.Wrap(err, "error during log initialization")
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

	fn("domain.default-limit", 30)

	fn("worker.enable", true)
	fn("worker.producer.spread.timeout", "10s")
	fn("worker.producer.spread.concurrency", 100)
	fn("worker.producer.spread.concurrency-output", 100)
	fn("worker.producer.delivery.timeout", "10s")
	fn("worker.producer.delivery.concurrency", 1000)

	fn("node.master.eligible", true)
	fn("node.master.election", "1m")
	fn("node.master.exclusive", false)
	fn("node.master.election-keep-alive", "30s")
	fn("node.worker.register", "1m")
	fn("node.worker.register-keep-alive", "30s")

	fn("provider.repository", "cassandra")
	fn("provider.queue", "aws.sqs")

	fn("provider.aws.sqs.producer.spread.queue", "flare-producer-spread")
	fn("provider.aws.sqs.producer.spread.ingress.timeout", "1s")
	fn("provider.aws.sqs.producer.spread.egress.receive-wait-time", "20s")
	fn("provider.aws.sqs.producer.delivery.queue", "flare-producer-delivery")
	fn("provider.aws.sqs.producer.delivery.ingress.timeout", "1s")
	fn("provider.aws.sqs.producer.delivery.egress.receive-wait-time", "20s")

	fn("provider.cassandra.hosts", []string{"127.0.0.1"})
	fn("provider.cassandra.port", 9042)
	fn("provider.cassandra.timeout", "600ms")
	fn("provider.cassandra.keyspace", "flare")
}
