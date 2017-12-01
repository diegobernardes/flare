// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongodb

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
)

// Client is used to interact with MongoDB.
type Client struct {
	addrs      []string
	database   string
	username   string
	password   string
	replicaSet string
	sess       *mgo.Session
	poolLimit  int
	timeout    time.Duration
}

// Stop close the session with MongoDB.
func (c *Client) Stop() {
	c.sess.Close()
}

func (c *Client) session() *mgo.Session { return c.sess.Clone() }

// NewClient returns a configured client to access MongoDB.
func NewClient(options ...func(*Client)) (*Client, error) {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	if len(c.addrs) == 0 {
		c.addrs = []string{"localhost:27017"}
	}

	if c.database == "" {
		c.database = "flare"
	}

	if c.poolLimit < 0 {
		return nil, fmt.Errorf("invalid pool limit '%d'", c.poolLimit)
	} else if c.poolLimit == 0 {
		c.poolLimit = 4096
	}

	if c.timeout == 0 {
		c.timeout = time.Second
	}

	di := &mgo.DialInfo{
		Addrs:          c.addrs,
		Database:       c.database,
		FailFast:       true,
		Username:       c.username,
		Password:       c.password,
		ReplicaSetName: c.replicaSet,
		PoolLimit:      c.poolLimit,
	}

	session, err := mgo.DialWithInfo(di)
	if err != nil {
		return nil, errors.Wrap(err, "error during connecting to MongoDB")
	}
	c.sess = session

	return c, nil
}

// ClientAddrs set the address to connect to MongoDB.
func ClientAddrs(addrs []string) func(*Client) {
	return func(c *Client) { c.addrs = addrs }
}

// ClientDatabase eset the database name to use.
func ClientDatabase(database string) func(*Client) {
	return func(c *Client) { c.database = database }
}

// ClientUsername set the username to authenticate.
func ClientUsername(username string) func(*Client) {
	return func(c *Client) { c.username = username }
}

// ClientPassword set the password to authenticate.
func ClientPassword(password string) func(*Client) {
	return func(c *Client) { c.password = password }
}

// ClientReplicaSet set the replicaSet.
func ClientReplicaSet(replicaSet string) func(*Client) {
	return func(c *Client) { c.replicaSet = replicaSet }
}

// ClientPoolLimit set the limit of connections per server.
func ClientPoolLimit(limit int) func(*Client) {
	return func(c *Client) { c.poolLimit = limit }
}

// ClientTimeout set the timeout during operations.
func ClientTimeout(timeout time.Duration) func(*Client) {
	return func(c *Client) { c.timeout = timeout }
}
