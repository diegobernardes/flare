package flare

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/external/cassandra"
	"github.com/diegobernardes/flare/infra/config"
)

const (
	externalCassandra = "cassandra"
	externalSQS       = "sqs"
)

type external struct {
	config          *config.Client
	cassandraClient *cassandra.Client
}

func (e *external) init() error {
	for _, source := range e.sources() {
		switch source {
		case externalCassandra:
			if err := e.initCassandra(); err != nil {
				return errors.Wrap(err, "error during external initialization")
			}
		case externalSQS:
		default:
			return fmt.Errorf("invalid source '%s'", source)
		}
	}

	return nil
}

func (e *external) start() error {
	for _, source := range e.sources() {
		switch source {
		case externalCassandra:
			if err := e.cassandraClient.Start(); err != nil {
				return errors.Wrap(err, "error during cassandra initialization")
			}
		case externalSQS:
		default:
			return fmt.Errorf("invalid source '%s'", source)
		}
	}

	return nil
}

func (e *external) stop() {
	e.cassandraClient.Stop()
}

func (e *external) initCassandra() error {
	e.cassandraClient = &cassandra.Client{
		Hosts:         e.config.GetStringSlice("external.cassandra.hosts"),
		Port:          e.config.GetInt("external.cassandra.port"),
		Keyspace:      e.config.GetString("external.cassandra.keyspace"),
		AvoidKeyspace: e.config.GetBool("external.cassandra.avoidKeyspace"),
	}

	if err := e.cassandraClient.Init(); err != nil {
		return errors.Wrap(err, "error during cassandra initialization")
	}
	return nil
}

func (e *external) sources() []string {
	var (
		result []string
		raw    = []string{
			e.config.GetString("election.source"),
			e.config.GetString("discovery.source"),
			e.config.GetString("repository.source"),
			e.config.GetString("queue.source"),
		}
	)
	fn := func(content string) bool {
		for _, r := range result {
			if r == content {
				return true
			}
		}
		return false
	}

	for _, r := range raw {
		if r == "" || fn(r) {
			continue
		}
		result = append(result, r)
	}

	return result
}
