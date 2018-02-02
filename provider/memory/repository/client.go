// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"github.com/diegobernardes/flare"
)

// Client implements the repository interface.
type Client struct {
	resource        Resource
	resourceOptions []func(*Resource)
	subscription    Subscription
	document        Document
}

// Resource return a resource repository.
func (c *Client) Resource() flare.ResourceRepositorier {
	return &c.resource
}

// Subscription return a subscription repository.
func (c *Client) Subscription() flare.SubscriptionRepositorier {
	return &c.subscription
}

// Document return a document repository.
func (c *Client) Document() flare.DocumentRepositorier {
	return &c.document
}

// NewClient return a configured client.
func NewClient(options ...func(*Client)) *Client {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	c.resource.repository = &c.subscription
	c.subscription.resourceRepository = &c.resource
	c.subscription.documentRepository = &c.document

	c.resource.init(c.resourceOptions...)
	c.subscription.init()
	c.document.init()
	return c
}

// ClientResourceOptions set the options to initialize the resource repository.
func ClientResourceOptions(options ...func(*Resource)) func(*Client) {
	return func(c *Client) {
		c.resourceOptions = options
	}
}
