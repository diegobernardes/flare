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

// CleanDocument implements the logic to process messages to delete the documents of a resource.
type CleanDocument struct {
	Repository flare.DocumentRepositorier
	Pusher     worker.Pusher
}

// Init check if the CleanDocument struct has everything it needs to run.
func (cd *CleanDocument) Init() error {
	if cd.Repository == nil {
		return errors.New("missing repository")
	}

	if cd.Pusher == nil {
		return errors.New("missing pusher")
	}
	return nil
}

// Process handle the stream of messages to be processed.
func (cd *CleanDocument) Process(ctx context.Context, content []byte) error {
	id, err := cd.unmarshal(content)
	if err != nil {
		return errors.Wrap(err, "error during message unmarshal")
	}

	if err := cd.Repository.DeleteByResourceID(ctx, id); err != nil {
		return errors.Wrapf(err, "error during delete all the documents from resource '%s'", id)
	}
	return nil
}

// Enqueue delivery the message to another stream to handle it.
func (cd *CleanDocument) Enqueue(ctx context.Context, id string) error {
	content, err := cd.marshal(id)
	if err != nil {
		return errors.Wrap(err, "error during message marshal")
	}

	if err := cd.Pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

func (*CleanDocument) marshal(id string) ([]byte, error) {
	rawContent := map[string]interface{}{"id": id}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal to json")
	}

	return content, nil
}

func (*CleanDocument) unmarshal(rawContent []byte) (string, error) {
	type message struct {
		ID string `json:"id"`
	}

	var m message
	if err := json.Unmarshal(rawContent, &m); err != nil {
		return "", errors.Wrap(err, "error during message unmarshal from json")
	}

	return m.ID, nil
}
