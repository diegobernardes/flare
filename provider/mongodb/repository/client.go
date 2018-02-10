// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

// Client that implements the repository interface.
type Client struct {
	base            *mongodb.Client
	resource        Resource
	resourceOptions []func(*Resource)

	subscription Subscription
	document     Document
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

// Stop the repository.
func (c *Client) Stop() error { return nil }

// Setup initialize the repository.
func (c *Client) Setup(_ context.Context) error {
	if err := c.resource.ensureIndex(); err != nil {
		return errors.Wrap(err, "error during resource index initialization")
	}

	if err := c.document.ensureIndex(); err != nil {
		return errors.Wrap(err, "error during document index initialization")
	}

	if err := c.subscription.ensureIndex(); err != nil {
		return errors.Wrap(err, "error during subscription index initialization")
	}

	return nil
}

// NewClient return a configured client to access the repositories.
func NewClient(options ...func(*Client)) (*Client, error) {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	if c.base == nil {
		return nil, errors.New("invalid MongoDB client")
	}

	c.resource.subscriptionRepository = &c.subscription
	c.subscription.resourceRepository = &c.resource
	c.subscription.documentRepository = &c.document
	c.resource.client = c.base
	c.subscription.client = c.base
	c.document.client = c.base

	if err := c.resource.init(c.resourceOptions...); err != nil {
		return nil, errors.Wrap(err, "error during resource repository initialization")
	}

	if err := c.subscription.init(); err != nil {
		return nil, errors.Wrap(err, "error during subscription repository initialization")
	}

	if err := c.document.init(); err != nil {
		return nil, errors.Wrap(err, "error during document repository initialization")
	}

	return c, nil
}

// ClientConnection set the MongoDB client.
func ClientConnection(base *mongodb.Client) func(*Client) {
	return func(c *Client) { c.base = base }
}

// ClientResourceOptions set the resource options.
func ClientResourceOptions(options ...func(*Resource)) func(*Client) {
	return func(c *Client) { c.resourceOptions = options }
}
