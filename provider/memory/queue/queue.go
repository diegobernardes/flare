// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"sync"
	"time"
)

// Client implements the queue interface.
type Client struct {
	mutex    sync.Mutex
	messages [][]byte
}

// Push the message to queue.
func (c *Client) Push(_ context.Context, content []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.messages = append(c.messages, content)
	return nil
}

// Pull fetch a message from queue.
func (c *Client) Pull(ctx context.Context, fn func(context.Context, []byte) error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.messages) == 0 {
		<-time.After(100 * time.Millisecond)
		return nil
	}

	if err := fn(ctx, c.messages[0]); err != nil {
		return err
	}
	c.messages = c.messages[1:]
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
