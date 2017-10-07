// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongodb

import (
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
)

// Client is used to interact with MongoDB.
type Client struct {
	addrs    []string
	database string
	username string
	password string
	sess     *mgo.Session
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

	di := &mgo.DialInfo{
		Addrs:    c.addrs,
		Database: c.database,
		FailFast: true,
		Username: c.username,
		Password: c.password,
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
