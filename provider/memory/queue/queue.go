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
	mutex          sync.Mutex
	messages       [][]byte
	timeoutProcess time.Duration
}

// Push the message to queue.
func (q *Client) Push(_ context.Context, content []byte) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.messages = append(q.messages, content)
	return nil
}

// Pull fetch a message from queue.
func (q *Client) Pull(ctx context.Context, fn func(context.Context, []byte) error) error {
	q.mutex.Lock()

	if len(q.messages) == 0 {
		q.mutex.Unlock()
		<-time.After(1 * time.Second)
		return nil
	}
	defer q.mutex.Unlock()

	ctx, ctxCancel := context.WithTimeout(ctx, q.timeoutProcess)
	defer ctxCancel()

	if err := fn(ctx, q.messages[0]); err != nil {
		return err
	}
	q.messages = q.messages[1:]
	return nil
}

// NewClient return a configured client.
func NewClient(options ...func(*Client)) *Client {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	if c.timeoutProcess == 0 {
		c.timeoutProcess = time.Hour
	}

	return c
}

// ClientProcessTimeout set the max duration a message has to be processed.
func ClientProcessTimeout(timeout time.Duration) func(*Client) {
	return func(c *Client) {
		c.timeoutProcess = timeout
	}
}
