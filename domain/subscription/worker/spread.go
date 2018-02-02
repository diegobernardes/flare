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

type spreadOutputer interface {
	Push(
		ctx context.Context, subscription *flare.Subscription, document *flare.Document, action string,
	) error
}

// Spread is used to process fetch all the subscriptions of a given partition and generate new
// messages to be processed.
type Spread struct {
	concurrency        int
	concurrencyControl chan struct{}
	repository         flare.SubscriptionRepositorier
	pusher             worker.Pusher
	output             spreadOutputer
}

// Push the signal to process the resource partition.
func (s *Spread) Push(
	ctx context.Context, document *flare.Document, action, partition string,
) error {
	content, err := s.marshal(document, action, partition)
	if err != nil {
		return errors.Wrap(err, "error during trigger")
	}

	if err = s.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

// Process the message.
func (s *Spread) Process(ctx context.Context, rawContent []byte) error {
	document, action, partition, err := s.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal the message")
	}

	chanSubscription, chanErr, err := s.repository.FindByPartition(
		ctx, document.Resource.ID, partition,
	)
	if err != nil {
		return errors.Wrap(err, "error during subscriptions find by partition")
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		for {
			select {
			case subscription, ok := <-chanSubscription:
				if !ok {
					return nil
				}

				if err := s.output.Push(groupCtx, &subscription, document, action); err != nil {
					return err
				}
			case err := <-chanErr:
				return err
			}
		}
	})

	if err := group.Wait(); err != nil {
		return errors.Wrap(err, "error during output")
	}
	return nil
}

// Init initialize the Spread.
func (s *Spread) Init(options ...func(*Spread)) error {
	for _, option := range options {
		option(s)
	}

	if s.concurrency < 0 {
		return errors.New("invalid concurrency")
	}
	if s.concurrency == 0 {
		s.concurrency = 1
	}
	s.concurrencyControl = make(chan struct{}, s.concurrency)

	if s.pusher == nil {
		return errors.New("pusher not found")
	}

	if s.repository == nil {
		return errors.New("repository not found")
	}

	if s.output == nil {
		return errors.New("output not found")
	}

	return nil
}

func (s *Spread) marshal(document *flare.Document, action, partition string) ([]byte, error) {
	rawContent := struct {
		Action     string `json:"action"`
		DocumentID string `json:"documentID"`
		ResourceID string `json:"resourceID"`
		Partition  string `json:"partition"`
	}{
		Action:     action,
		DocumentID: document.ID,
		ResourceID: document.Resource.ID,
		Partition:  partition,
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (s *Spread) unmarshal(rawContent []byte) (*flare.Document, string, string, error) {
	type content struct {
		Action     string `json:"action"`
		DocumentID string `json:"documentID"`
		ResourceID string `json:"resourceID"`
		Partition  string `json:"partition"`
	}

	var value content
	if err := json.Unmarshal(rawContent, &value); err != nil {
		return nil, "", "", errors.Wrap(err, "error during content unmarshal")
	}

	return &flare.Document{
		ID:       value.DocumentID,
		Resource: flare.Resource{ID: value.ResourceID},
	}, value.Action, value.Partition, nil
}

// SpreadSubscriptionRepository set the subscription repository.
func SpreadSubscriptionRepository(repository flare.SubscriptionRepositorier) func(*Spread) {
	return func(s *Spread) { s.repository = repository }
}

// SpreadPusher set the output of the messages.
func SpreadPusher(pusher worker.Pusher) func(*Spread) {
	return func(s *Spread) { s.pusher = pusher }
}

// SpreadConcurrency set the concurrency to send the output result.
func SpreadConcurrency(concurrency int) func(*Spread) {
	return func(s *Spread) { s.concurrency = concurrency }
}

// SpreadOutput set the output of the result.
func SpreadOutput(output spreadOutputer) func(*Spread) {
	return func(s *Spread) { s.output = output }
}
