// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"

	"github.com/pkg/errors"
)

// WorkerWrap is used with Worker to implement a generic worker. The messages should be
// encapsulated with a metadata that informs who can process the message, and this struct is for
// this.
type WorkerWrap struct {
	Worker *Worker
	Task   string
}

// Init check if WorkerWrap has everything it needs to run.
func (ww *WorkerWrap) Init() error {
	if ww.Worker == nil {
		return errors.New("missing worker")
	}

	if ww.Task == "" {
		return errors.New("missing task")
	}
	return nil
}

// Push send the message to be processed by the worker.
func (ww *WorkerWrap) Push(ctx context.Context, content []byte) error {
	return ww.Worker.Enqueue(ctx, content, ww.Task)
}
