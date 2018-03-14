// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/scheduler/cluster"
	"github.com/diegobernardes/flare/scheduler/election"
	"github.com/diegobernardes/flare/scheduler/node"
)

// Client implements the logic to schedule the task among the cluster.
type Client struct {
	Cluster            *cluster.Client
	Election           *election.Client
	ConsumerDispatcher *ConsumerDispatcher
	node               node.Node
}

// Start the scheduler.
func (c *Client) Start() {
	c.Cluster.Start()
	c.Election.Start()

	go func() {
		<-time.After(10 * time.Second)
		c.ConsumerDispatcher.start()
	}()
}

// Stop the scheduler.
func (c *Client) Stop() {
	c.Cluster.Stop()
	c.Election.Stop()
}

// Init is used to initialize the scheduler.
func (c *Client) Init() error {
	c.node.Init()

	if c.Cluster == nil {
		return errors.New("missing Cluster")
	}
	c.Cluster.NodeID = c.node.ID

	if err := c.Cluster.Init(); err != nil {
		return errors.Wrap(err, "error during Cluster initialization")
	}

	if c.Election == nil {
		return errors.New("missing Election")
	}

	c.Election.NodeID = c.node.ID
	if err := c.Election.Init(); err != nil {
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
