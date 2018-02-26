// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"time"

	"github.com/pkg/errors"
)

// Client implements the logic to schedule the task among the cluster.
type Client struct {
	Manager            *Manager
	Election           *Election
	ConsumerDispatcher *ConsumerDispatcher
	node               Node
}

// Start the scheduler.
func (c *Client) Start() {
	c.Manager.start()
	c.Election.start()

	go func() {
		<-time.After(10 * time.Second)
		c.ConsumerDispatcher.start()
	}()
}

// Stop the scheduler.
func (c *Client) Stop() {
	c.Manager.stop()
	c.Election.stop()
}

// Init is used to initialize the scheduler.
func (c *Client) Init() error {
	c.node.init()

	if c.Manager == nil {
		return errors.New("missing Manager")
	}
	c.Manager.nodeID = c.node.ID

	if err := c.Manager.init(); err != nil {
		return errors.Wrap(err, "error during Manager initialization")
	}

	if c.Election == nil {
		return errors.New("missing Election")
	}
	c.Election.nodeID = c.node.ID

	if err := c.Election.init(); err != nil {
		return errors.Wrap(err, "error during Election initialization")
	}

	if c.ConsumerDispatcher == nil {
		return errors.New("missing ConsumerDispatcher")
	}
	c.ConsumerDispatcher.nodeID = c.node.ID

	if err := c.ConsumerDispatcher.init(); err != nil {
		return errors.New("error during ConsumerDispatcher initialization")
	}

	return nil
}
