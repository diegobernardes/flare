// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/pkg/errors"
)

// Pusher is used to send a task to be processed.
type Pusher interface {
	Push(context.Context, []byte) error
}

// Puller is used to fetch a task to process.
type Puller interface {
	Pull(context.Context, func(context.Context, []byte) error) error
}

// Processor is used to process the tasks.
type Processor interface {
	Process(context.Context, []byte) error
}

// Client implements the logic to process tasks.
type Client struct {
	pusher     Pusher
	puller     Puller
	processor  Processor
	goroutines int
	ctx        context.Context
	ctxCancel  func()
	logger     log.Logger
	timeout    time.Duration
	wg         sync.WaitGroup
}

// Push the task to be processed.
func (c *Client) Push(ctx context.Context, content []byte) error {
	return errors.Wrap(c.pusher.Push(ctx, content), "error during task push")
}

// Start the worker to process tasks.
func (c *Client) Start() {
	c.wg.Add(c.goroutines)
	for i := 0; i < c.goroutines; i++ {
		go func() {
			defer c.wg.Done()

			for {
				c.process()

				if err := c.ctx.Err(); err != nil {
					break
				}
			}
		}()
	}
}

// Stop the client.
func (c *Client) Stop() {
	c.ctxCancel()
	c.wg.Wait()
}

func (c *Client) process() {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(c.logger).Log("message", "panic during worker process", "reason", err)
		}
	}()

	ctx, ctxCancel := context.WithTimeout(c.ctx, c.timeout)
	defer ctxCancel()

	err := c.puller.Pull(ctx, func(fnCtx context.Context, content []byte) error {
		level.Info(c.logger).Log("message", "message received to be processed")

		if err := c.processor.Process(fnCtx, content); err != nil {
			level.Error(c.logger).Log("error", err.Error(), "message", "error during message process")
			return err
		}
		return nil
	})
	if err != nil {
		level.Error(c.logger).Log("error", err.Error(), "message", "error during message pull")
	}
}

// NewClient returns a configured worker.
func NewClient(options ...func(*Client)) (*Client, error) {
	c := &Client{}

	for _, option := range options {
		option(c)
	}

	if c.pusher == nil {
		return nil, errors.New("pusher not found")
	}

	if c.puller == nil {
		return nil, errors.New("puller not found")
	}

	if c.processor == nil {
		return nil, errors.New("processor not found")
	}

	if c.goroutines < 0 {
		return nil, errors.New("invalid goroutines count")
	}

	if c.logger == nil {
		return nil, errors.New("logger not found")
	}
	c.logger = log.With(c.logger, "package", "infra/task")

	if c.timeout == 0 {
		return nil, errors.New("invalid timeout")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.ctxCancel = ctxCancel

	return c, nil
}

// WorkerPusher set the pusher at worker.
func WorkerPusher(pusher Pusher) func(*Client) {
	return func(c *Client) { c.pusher = pusher }
}

// WorkerPuller set the puller at worker.
func WorkerPuller(puller Puller) func(*Client) {
	return func(c *Client) { c.puller = puller }
}

// WorkerProcessor set the processor at Worker.
func WorkerProcessor(processor Processor) func(*Client) {
	return func(c *Client) { c.processor = processor }
}

// WorkerGoroutines set the quantity of goroutines used to process the queue.
func WorkerGoroutines(goroutines int) func(*Client) {
	return func(c *Client) { c.goroutines = goroutines }
}

// WorkerLogger set the worker logger.
func WorkerLogger(logger log.Logger) func(*Client) {
	return func(c *Client) { c.logger = logger }
}

// WorkerTimeout set the default timeout duration for a task to process.
func WorkerTimeout(timeout time.Duration) func(*Client) {
	return func(c *Client) { c.timeout = timeout }
}
