// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"runtime"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
)

// Variables set with ldflags during compilation.
var (
	Version   = ""
	BuildTime = ""
	Commit    = ""
	GoVersion = runtime.Version()
)

var (
	providerMemory  = "memory"
	providerAWSSQS  = "aws.sqs"
	providerMongoDB = "mongodb"
)

// Client is responsible to init Flare.
type Client struct {
	logger     log.Logger
	config     *config.Client
	repository *repository
	queue      *queue
	server     *server
	worker     *worker
	domain     *domain
	hook       *hook
}

// Start the service.
func (c *Client) Start() error {
	level.Info(c.logger).Log("message", "starting service")

	c.worker.cfg = c.config
	c.worker.repository = c.repository
	c.worker.queue = c.queue
	c.worker.logger = c.logger
	c.worker.hook = c.hook
	if err := c.worker.init(); err != nil {
		return errors.Wrap(err, "error during worker initialization")
	}

	c.domain.logger = c.logger
	c.domain.repository = c.repository
	c.domain.worker = c.worker
	c.domain.cfg = c.config
	c.domain.hook = c.hook
	if err := c.domain.init(); err != nil {
		return errors.Wrap(err, "error during domain initialization")
	}

	c.server.cfg = c.config
	c.server.logger = c.logger
	c.server.handler.resource = c.domain.resource
	c.server.handler.subscription = c.domain.subscription
	c.server.handler.document = c.domain.document
	if err := c.server.init(); err != nil {
		return errors.Wrap(err, "error during server initialization")
	}

	level.Info(c.logger).Log("message", "service initialized")
	return nil
}

// Stop the service.
func (c *Client) Stop() error {
	level.Info(c.logger).Log("message", "signal to close the process received")
	level.Info(c.logger).Log("message", "closing the server")
	if err := c.server.stop(); err != nil {
		return errors.Wrap(err, "error during server stop")
	}

	level.Info(c.logger).Log("message", "closing the worker")
	if err := c.worker.stop(); err != nil {
		return errors.Wrap(err, "error during worker stop")
	}

	level.Info(c.logger).Log("message", "closing the repository")
	if err := c.repository.stop(); err != nil {
		return errors.Wrap(err, "error during repository stop")
	}

	level.Info(c.logger).Log("message", "bye!")
	return nil
}

// Setup is used to bootstrap the providers.
func (c *Client) Setup(ctx context.Context) error {
	level.Info(c.logger).Log("message", "starting repository setup")
	if err := c.repository.setup(ctx); err != nil {
		return errors.Wrap(err, "error during repository setup")
	}
	level.Info(c.logger).Log("message", "repository setup done")

	level.Info(c.logger).Log("message", "starting queue setup")
	if err := c.queue.setup(ctx); err != nil {
		return errors.Wrap(err, "error during queue initialization")
	}
	level.Info(c.logger).Log("message", "queue setup done")

	return nil
}

func (c *Client) init() error {
	if err := c.config.Init(); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}
	c.loadDefaultValues()

	if err := c.initLogger(); err != nil {
		return errors.Wrap(err, "error during log initialization")
	}

	c.hook.init()

	c.repository.cfg = c.config
	if err := c.repository.init(); err != nil {
		return errors.Wrap(err, "error during repository initialization")
	}

	c.queue.cfg = c.config
	c.queue.logger = c.logger
	if err := c.queue.init(); err != nil {
		return errors.Wrap(err, "errors during queue initialization")
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

	fn("domain.resource.partition", 1000)
	fn("domain.pagination.default-limit", 30)

	fn("worker.enable", true)
	fn("worker.subscription.partition.timeout", "10s")
	fn("worker.subscription.partition.concurrency", 10)
	fn("worker.subscription.partition.concurrency-output", 100)
	fn("worker.subscription.spread.timeout", "10s")
	fn("worker.subscription.spread.concurrency", 10)
	fn("worker.subscription.spread.concurrency-output", 100)
	fn("worker.subscription.delivery.timeout", "10s")
	fn("worker.subscription.delivery.concurrency", 10)
	fn("worker.generic.timeout", "180s")
	fn("worker.generic.concurrency", 100)

	fn("provider.repository", "memory")
	fn("provider.queue", "memory")

	fn("provider.aws.sqs.queue.subscription.partition.queue", "flare-subscription-partition")
	fn("provider.aws.sqs.queue.subscription.partition.ingress.timeout", "1s")
	fn("provider.aws.sqs.queue.subscription.partition.egress.receive-wait-time", "20s")
	fn("provider.aws.sqs.queue.subscription.spread.queue", "flare-subscription-spread")
	fn("provider.aws.sqs.queue.subscription.spread.ingress.timeout", "1s")
	fn("provider.aws.sqs.queue.subscription.spread.egress.receive-wait-time", "20s")
	fn("provider.aws.sqs.queue.subscription.delivery.queue", "flare-subscription-delivery")
	fn("provider.aws.sqs.queue.subscription.delivery.ingress.timeout", "1s")
	fn("provider.aws.sqs.queue.subscription.delivery.egress.receive-wait-time", "20s")
	fn("provider.aws.sqs.queue.generic.queue", "flare-generic")
	fn("provider.aws.sqs.queue.generic.ingress.timeout", "1s")
	fn("provider.aws.sqs.queue.generic.egress.receive-wait-time", "20s")

	fn("provider.mongodb.addrs", []string{"localhost:27017"})
	fn("provider.mongodb.database", "flare")
	fn("provider.mongodb.pool-limit", 4096)
	fn("provider.mongodb.timeout", "1s")

}

// NewClient return a initialized client.
func NewClient(options ...func(*Client)) (*Client, error) {
	c := &Client{
		config:     &config.Client{},
		repository: &repository{},
		queue:      &queue{},
		server:     &server{},
		worker:     &worker{},
		domain:     &domain{},
		hook:       &hook{},
	}

	for _, option := range options {
		option(c)
	}

	if err := c.init(); err != nil {
		return nil, errors.Wrap(err, "error during client initialization")
	}

	return c, nil
}

// ClientConfig set the config on client.
func ClientConfig(config string) func(*Client) {
	return func(c *Client) { c.config.Content = config }
}
