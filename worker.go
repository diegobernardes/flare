// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/worker"
)

// Worker act as a queue that receive the messages from other workers and as a dispatcher as they
// process the requests from the same queue and deliver it to other workers.
type Worker struct {
	Logger log.Logger
	Pusher worker.Pusher

	tasks map[string]worker.Processor
}

// Init validate if the worker has everything it needs to run.
func (w *Worker) Init() error {
	if w.Logger == nil {
		return errors.New("missing logger")
	}

	if w.Pusher == nil {
		return errors.New("missing pusher")
	}

	w.tasks = make(map[string]worker.Processor)
	return nil
}

// Process the message.
func (w *Worker) Process(ctx context.Context, rawContent []byte) error {
	task, payload, err := w.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "error during marshal content to json")
	}

	processor, ok := w.tasks[task]
	if !ok {
		level.Info(w.Logger).Log("message", "ignoring message as processor is not found", "task", task)
		return nil
	}

	if err := processor.Process(ctx, payload); err != nil {
		return errors.Wrap(err, "error during process")
	}
	return nil
}

// Enqueue the message to process it later.
func (w *Worker) Enqueue(ctx context.Context, rawContent []byte, task string) error {
	content, err := w.marshal(task, rawContent)
	if err != nil {
		panic(err)
	}

	if err := w.Pusher.Push(ctx, content); err != nil {
		panic(err)
	}
	return nil
}

// Register a processor for a given task.
func (w *Worker) Register(task string, processor worker.Processor) error {
	if _, ok := w.tasks[task]; ok {
		return fmt.Errorf("already have processor associated with the task '%s'", task)
	}

	w.tasks[task] = processor
	return nil
}

func (w *Worker) marshal(task string, rawContent []byte) ([]byte, error) {
	type protocol struct {
		Task    string          `json:"task"`
		Payload json.RawMessage `json:"payload"`
	}

	message := protocol{Task: task, Payload: rawContent}
	content, err := json.Marshal(&message)
	if err != nil {
		panic(err)
	}
	return content, nil
}

func (w *Worker) unmarshal(content []byte) (string, []byte, error) {
	type protocol struct {
		Task    string          `json:"task"`
		Payload json.RawMessage `json:"payload"`
	}

	var message protocol
	if err := json.Unmarshal(content, &message); err != nil {
		panic(err)
	}

	return message.Task, message.Payload, nil
}
