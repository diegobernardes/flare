package flare

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/external/aws"
	"github.com/diegobernardes/flare/external/etcd"
	"github.com/diegobernardes/flare/infra/config"
)

const (
	externalEtcd = "etcd"
	externalSQS  = "sqs"
)

type external struct {
	config     *config.Client
	etcdClient *etcd.Client
	awsClient  *aws.Client
}

func (e *external) init() error {
	for _, source := range e.sources() {
		switch source {
		case externalEtcd:
			if err := e.initEtcd(); err != nil {
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
		case externalEtcd:
			if err := e.etcdClient.Start(); err != nil {
				return errors.Wrap(err, "error during etcd initialization")
			}
		case externalSQS:
		default:
			return fmt.Errorf("invalid source '%s'", source)
		}
	}

	return nil
}

func (e *external) stop() error {
	return errors.Wrap(e.etcdClient.Stop(), "error during etcd.Client stop")
}

func (e *external) initEtcd() error {
	timeout, err := e.config.GetDuration("external.etcd.dial-timeout")
	if err != nil {
		return errors.Wrap(err, "error during load 'external.etcd.dial-timeout' config")
	}

	e.etcdClient = &etcd.Client{
		Username:    e.config.GetString("external.etcd.username"),
		Password:    e.config.GetString("external.etcd.password"),
		Endpoints:   e.config.GetStringSlice("external.etcd.addr"),
		DialTimeout: timeout,
	}

	if err := e.etcdClient.Init(); err != nil {
		return errors.Wrap(err, "error during etcd initialization")
	}
	return nil
}

func (e *external) sources() []string {
	var (
		result []string
		raw    = []string{
			e.config.GetString("consumer.state-source"),
			e.config.GetString("cluster.source"),
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
