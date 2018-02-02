// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import "context"

// Client that implements the queue interface.
type Client struct {
	Content []byte
	err     error
}

// Push the content to the queue.
func (c *Client) Push(ctx context.Context, content []byte) error {
	if c.err != nil {
		return c.err
	}
	c.Content = content
	return nil
}

// NewClient return a configured client.
func NewClient(options ...func(*Client)) *Client {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	return c
}

// ClientError set the error to be returned during operations.
func ClientError(err error) func(*Client) {
	return func(c *Client) {
		c.err = err
	}
}
