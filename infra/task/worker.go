// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package task

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/pkg/errors"
)

// Worker implements the logic to process tasks.
type Worker struct {
	pusher         Pusher
	puller         Puller
	processor      Processor
	timeoutProcess time.Duration
	timeoutPush    time.Duration
	goroutines     int
	ctx            context.Context
	ctxCancel      func()
	logger         log.Logger
}

// Push the task to be processed.
func (w *Worker) Push(ctx context.Context, content []byte) error {
	return errors.Wrap(w.pusher.Push(ctx, content), "error during task push")
}

// Start the worker to process tasks.
func (w *Worker) Start() {
	for i := 0; i < w.goroutines; i++ {
		go func() {
			for {
				w.process()

				if err := w.ctx.Err(); err != nil {
					break
				}
			}
		}()
	}
}

func (w *Worker) process() {
	defer func() { recover() }()

	ctx, ctxCancel := context.WithTimeout(w.ctx, w.timeoutProcess)
	defer ctxCancel()

	errLogger := level.Error(w.logger)
	w.puller.Pull(ctx, func(ctx context.Context, content []byte) error {
		w.logger.Log("message", "new message received to be processed")
		err := w.processor.Process(ctx, content)
		if err != nil {
			errLogger.Log("error", err.Error(), "message", "error during message process")
			return err
		}
		return nil
	})
}

// NewWorker returns a configured worker.
func NewWorker(options ...func(*Worker)) (*Worker, error) {
	w := &Worker{}

	for _, option := range options {
		option(w)
	}

	if w.pusher == nil {
		return nil, errors.New("pusher not found")
	}

	if w.puller == nil {
		return nil, errors.New("puller not found")
	}

	if w.processor == nil {
		return nil, errors.New("processor not found")
	}

	if w.timeoutProcess == 0 {
		return nil, errors.New("invalid timeoutProcess")
	}

	if w.timeoutPush == 0 {
		return nil, errors.New("invalid timeoutPush")
	}

	if w.goroutines <= 0 {
		return nil, errors.New("invalid goroutines count")
	}

	if w.logger == nil {
		return nil, errors.New("logger not found")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	w.ctx = ctx
	w.ctxCancel = ctxCancel

	return w, nil
}

// WorkerPusher set the pusher at worker.
func WorkerPusher(pusher Pusher) func(*Worker) {
	return func(w *Worker) {
		w.pusher = pusher
	}
}

// WorkerPuller set the puller at worker.
func WorkerPuller(puller Puller) func(*Worker) {
	return func(w *Worker) {
		w.puller = puller
	}
}

// WorkerProcessor set the processor at Worker.
func WorkerProcessor(processor Processor) func(*Worker) {
	return func(w *Worker) {
		w.processor = processor
	}
}

// WorkerTimeoutProcess set the timeout to process the messages.
func WorkerTimeoutProcess(timeout time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.timeoutProcess = timeout
	}
}

// WorkerTimeoutPush set the timeout to push the message.
func WorkerTimeoutPush(timeout time.Duration) func(*Worker) {
	return func(w *Worker) {
		w.timeoutPush = timeout
	}
}

// WorkerGoroutines set the quantity of goroutines used to process the queue.
func WorkerGoroutines(goroutines int) func(*Worker) {
	return func(w *Worker) {
		w.goroutines = goroutines
	}
}

// WorkerLogger set the worker logger.
func WorkerLogger(logger log.Logger) func(*Worker) {
	return func(w *Worker) {
		w.logger = log.With(logger, "package", "infra/task")
	}
}
