// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/worker"
)

type partitionOutputer interface {
	Push(ctx context.Context, document *flare.Document, action, partition string) error
}

// Partition is used to process the signals on documents change.
type Partition struct {
	concurrency        int
	concurrencyControl chan struct{}
	repository         flare.ResourceRepositorier
	pusher             worker.Pusher
	output             partitionOutputer
}

func (t *Partition) marshal(document *flare.Document, action string) ([]byte, error) {
	rawContent := map[string]interface{}{
		"action":     action,
		"documentID": document.ID,
		"resourceID": document.Resource.ID,
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (t *Partition) unmarshal(rawContent []byte) (*flare.Document, string, error) {
	type content struct {
		Action     string `json:"action"`
		DocumentID string `json:"documentID"`
		ResourceID string `json:"resourceID"`
	}

	var value content
	if err := json.Unmarshal(rawContent, &value); err != nil {
		return nil, "", errors.Wrap(err, "error during parse content to json")
	}

	return &flare.Document{
		ID: value.DocumentID,
		Resource: flare.Resource{
			ID: value.ResourceID,
		},
	}, value.Action, nil
}

// Push the document change signal.
func (t *Partition) Push(ctx context.Context, document *flare.Document, action string) error {
	content, err := t.marshal(document, action)
	if err != nil {
		return errors.Wrap(err, "error during trigger")
	}

	if err = t.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

// Process is used to consume the tasks.
func (t *Partition) Process(ctx context.Context, rawContent []byte) error {
	document, action, err := t.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal the message")
	}

	partitions, err := t.repository.Partitions(ctx, document.Resource.ID)
	if err != nil {
		return errors.Wrap(err, "could not get the partition count of a resource")
	}

	group, groupCtx := errgroup.WithContext(ctx)
	for _, partition := range partitions {
		t.concurrencyControl <- struct{}{}

		group.Go(func(p string) func() error {
			return func() error {
				defer func() { <-t.concurrencyControl }()
				return t.output.Push(groupCtx, document, action, p)
			}
		}(partition))
	}

	if err := group.Wait(); err != nil {
		return errors.Wrap(err, "error during output")
	}
	return nil
}

// Init initialize the Partition.
func (t *Partition) Init(options ...func(*Partition)) error {
	for _, option := range options {
		option(t)
	}

	if t.concurrency < 0 {
		return errors.New("invalid concurrency")
	}
	if t.concurrency == 0 {
		t.concurrency = 1
	}
	t.concurrencyControl = make(chan struct{}, t.concurrency)

	if t.repository == nil {
		return errors.New("repository not found")
	}

	if t.pusher == nil {
		return errors.New("pusher not found")
	}

	if t.output == nil {
		return errors.New("output not found")
	}

	return nil
}

// PartitionResourceRepository set the repository on Trigger.
func PartitionResourceRepository(repository flare.ResourceRepositorier) func(*Partition) {
	return func(t *Partition) { t.repository = repository }
}

// PartitionPusher set the pusher that gonna receive the trigger notifications.
func PartitionPusher(pusher worker.Pusher) func(*Partition) {
	return func(t *Partition) { t.pusher = pusher }
}

// PartitionConcurrency control the concurrency used to output the result.
func PartitionConcurrency(concurrency int) func(*Partition) {
	return func(p *Partition) { p.concurrency = concurrency }
}

// PartitionOutput is used to receive the output from Partition worker.
func PartitionOutput(output partitionOutputer) func(*Partition) {
	return func(p *Partition) { p.output = output }
}
