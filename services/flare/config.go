// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/aws"
	"github.com/diegobernardes/flare/repository/memory"
	"github.com/diegobernardes/flare/repository/mongodb"
)

const (
	engineMemory  = "memory"
	engineMongoDB = "mongodb"
)

type config struct {
	content string
	viper   *viper.Viper

	subscription *mongodb.Subscription
	resource     *mongodb.Resource
}

func (c *config) getString(key string) string { return c.viper.GetString(key) }

func (c *config) getStringSlice(key string) []string { return c.viper.GetStringSlice(key) }

func (c *config) getInt(key string) int { return c.viper.GetInt(key) }

func (c *config) getDuration(key string) (time.Duration, error) {
	value := c.getString(key)
	if value == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("error during parse '%s' to time.Duration", key))
	}
	return duration, nil
}

func (c *config) documentRepository() (flare.DocumentRepositorier, error) {
	engine := c.getString("repository.engine")
	switch engine {
	case engineMongoDB:
		client, err := c.mongodb()
		if err != nil {
			return nil, err
		}

		repository, err := mongodb.NewDocument(mongodb.DocumentClient(client))
		if err != nil {
			return nil, err
		}
		return repository, nil
	case engineMemory:
		return memory.NewDocument(), nil
	default:
		return nil, fmt.Errorf("invalid repository.engine '%s'", engine)
	}
}

func (c *config) subscriptionRepository() (flare.SubscriptionRepositorier, error) {
	engine := c.getString("repository.engine")
	switch engine {
	case engineMongoDB:
		client, err := c.mongodb()
		if err != nil {
			return nil, err
		}

		if err = c.subscription.Init(mongodb.SubscriptionClient(client)); err != nil {
			return nil, err
		}
		return c.subscription, nil
	case engineMemory:
		return memory.NewSubscription(), nil
	default:
		return nil, fmt.Errorf("invalid repository.engine '%s'", engine)
	}
}

func (c *config) resourceRepository() (flare.ResourceRepositorier, error) {
	engine := c.getString("repository.engine")
	switch engine {
	case engineMongoDB:
		client, err := c.mongodb()
		if err != nil {
			return nil, err
		}

		if err = c.resource.Init(mongodb.ResourceClient(client)); err != nil {
			return nil, err
		}
		return c.resource, nil
	case engineMemory:
		return memory.NewResource(), nil
	default:
		return nil, fmt.Errorf("invalid repository.engine '%s'", engine)
	}
}

func (c *config) mongodb() (*mongodb.Client, error) {
	timeout, err := c.getDuration("repository.timeout")
	if err != nil {
		return nil, err
	}

	client, err := mongodb.NewClient(
		mongodb.ClientAddrs(c.getStringSlice("repository.addrs")),
		mongodb.ClientDatabase(c.getString("repository.database")),
		mongodb.ClientUsername(c.getString("repository.username")),
		mongodb.ClientPassword(c.getString("repository.password")),
		mongodb.ClientReplicaSet(c.getString("repository.replica-set")),
		mongodb.ClientPoolLimit(c.getInt("repository.pool-limit")),
		mongodb.ClientTimeout(timeout),
	)
	return client, errors.Wrap(err, "error during MongoDB connection")
}

func (c *config) sqsOptions(name string) ([]func(*aws.SQS), error) {
	session, err := aws.NewSession(
		aws.SessionKey(c.getString("aws.key")),
		aws.SessionSecret(c.getString("aws.secret")),
		aws.SessionRegion(c.getString("aws.region")),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during AWS session initialization")
	}

	return []func(*aws.SQS){
		aws.SQSQueueName(c.getString(fmt.Sprintf("task.queue-%s", name))),
		aws.SQSSession(session),
	}, nil
}

func (c *config) queue(name string) (*aws.SQS, *aws.SQS, error) {
	engine := c.getString("task.engine")
	if engine != "sqs" {
		return nil, nil, fmt.Errorf("invalid task.engine '%s'", engine)
	}

	options, err := c.sqsOptions(name)
	if err != nil {
		return nil, nil, err
	}

	sqs, err := aws.NewSQS(options...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error during AWS SQS initialization")
	}
	return sqs, sqs, nil
}

func (c *config) httpDefaultLimit() int {
	value := c.getInt("http.default-limit")
	if value == 0 {
		return 30
	}
	return value
}

func (c *config) serverMiddlewareTimeout() (time.Duration, error) {
	s := c.getString("http.timeout")
	if s == "" {
		s = "1s"
	}
	return time.ParseDuration(s)
}

func newConfig(options ...func(*config)) (*config, error) {
	c := &config{viper: viper.New()}
	c.viper.SetConfigType("toml")

	for _, option := range options {
		option(c)
	}

	if err := c.viper.ReadConfig(bytes.NewBufferString(c.content)); err != nil {
		return nil, errors.Wrap(err, "error during config setup")
	}

	if c.getString("repository.engine") == engineMongoDB {
		c.resource = &mongodb.Resource{}
		c.subscription = &mongodb.Subscription{}
		c.resource.SetSubscriptionRepository(c.subscription)
		c.subscription.SetResourceRepository(c.resource)
	}

	return c, nil
}

func configContent(content string) func(*config) {
	return func(c *config) { c.content = content }
}
