// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/worker"
)

type queuer interface {
	Push(context.Context, []byte) error
	Pull(context.Context, func(context.Context, []byte) error) error
}

type queueCreator interface {
	Create(ctx context.Context, id string) error
}

type CreateQueue struct {
	Pusher  worker.Pusher
	Creator queueCreator
}

func (cq *CreateQueue) Init() error {
	if cq.Pusher == nil {
		return errors.New("missing pusher")
	}

	if cq.Creator == nil {
		return errors.New("missing creator")
	}
	return nil
}

func (cq *CreateQueue) Enqueue(ctx context.Context, s *flare.Subscription) error {
	content, err := cq.marshal(s)
	if err != nil {
		panic(err)
	}

	if err := cq.Pusher.Push(ctx, content); err != nil {
		panic(err)
	}
	return nil
}

func (cq *CreateQueue) Process(ctx context.Context, rawContent []byte) error {
	id, err := cq.unmarshal(rawContent)
	if err != nil {
		panic(err)
	}

	if err := cq.Creator.Create(ctx, id); err != nil {
		return err
	}
	return nil
}

func (cq *CreateQueue) marshal(s *flare.Subscription) ([]byte, error) {
	rawContent := struct {
		ID string `json:"id"`
	}{
		ID: s.ID,
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (cq *CreateQueue) unmarshal(rawContent []byte) (string, error) {
	type content struct {
		ID string `json:"id"`
	}

	var value content
	if err := json.Unmarshal(rawContent, &value); err != nil {
		return "", errors.Wrap(err, "error during content unmarshal")
	}

	return value.ID, nil
}
