package etcd

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
)

type Client struct {
	Endpoints   []string
	DialTimeout time.Duration

	base      *clientv3.Client
	lease     *lease
	leaseID   clientv3.LeaseID
	ctx       context.Context
	ctxCancel func()
}

func (c *Client) Init() error {
	if len(c.Endpoints) == 0 {
		return errors.New("missing Endpoints")
	}

	if c.DialTimeout < 0 {
		return errors.New("invalid DialTimeout")
	}

	return nil
}

func (c *Client) Start() error {
	base, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: c.DialTimeout,
	})
	if err != nil {
		return errors.Wrap(err, "error during etcd connection")
	}
	c.base = base

	c.lease.new(c.ctx)

	return nil
}
