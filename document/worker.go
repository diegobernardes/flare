// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/task"
)

// Worker is used to async process all the create, update and delete operations on documents.
type Worker struct {
	pusher              task.Pusher
	documentRepository  flare.DocumentRepositorier
	subscriptionTrigger flare.SubscriptionTrigger
}

// Process process the enqueued documents.
func (w *Worker) Process(ctx context.Context, rawContent []byte) error {
	action, doc, err := w.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "error during message unmarshal")
	}

	switch action {
	case flare.SubscriptionTriggerUpdate:
		if err := w.documentRepository.Update(ctx, doc); err != nil {
			return errors.Wrap(err, "error during document upsert")
		}

		if err := w.subscriptionTrigger.Update(ctx, doc); err != nil {
			return errors.Wrap(err, "error during document change trigger")
		}
	case flare.SubscriptionTriggerDelete:
		if err := w.subscriptionTrigger.Delete(ctx, doc); err != nil {
			return errors.Wrap(err, "error during document change trigger")
		}
	}
	return nil
}

func (w *Worker) push(ctx context.Context, action string, doc *flare.Document) error {
	content, err := w.marshal(action, doc)
	if err != nil {
		return errors.Wrap(err, "error during message marshal")
	}

	if err = w.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message push to be processed")
	}
	return nil
}

func (w *Worker) marshal(action string, doc *flare.Document) ([]byte, error) {
	resource := map[string]interface{}{
		"id": doc.Resource.ID,
	}

	document := map[string]interface{}{
		"id":        doc.ID,
		"updatedAt": doc.UpdatedAt,
	}

	message := map[string]interface{}{
		"action":   action,
		"document": document,
		"resource": resource,
	}

	if action == flare.SubscriptionTriggerUpdate {
		if doc.Resource.Change.Kind == flare.ResourceChangeDate {
			resource["dateFormat"] = doc.Resource.Change.DateFormat
		}

		resource["kind"] = doc.Resource.Change.Kind
		document["content"] = doc.Content
		document["revision"] = doc.Revision
	}

	content, err := json.Marshal(message)
	if err != nil {
		return nil, errors.Wrap(err, "error during json marshal")
	}
	return content, nil
}

func (w *Worker) unmarshal(rawContent []byte) (string, *flare.Document, error) {
	type message struct {
		Action   string `json:"action"`
		Document struct {
			ID        string                 `json:"id"`
			Revision  int64                  `json:"revision"`
			UpdatedAt time.Time              `json:"updatedAt"`
			Content   map[string]interface{} `json:"content"`
		} `json:"document"`
		Resource struct {
			ID         string `json:"id"`
			Kind       string `json:"kind"`
			DateFormat string `json:"dateFormat"`
		} `json:"resource"`
	}

	msg := &message{}
	if err := json.Unmarshal(rawContent, msg); err != nil {
		return "", nil, errors.Wrap(err, "error during json unmarshal")
	}

	return msg.Action, &flare.Document{
		ID:        msg.Document.ID,
		Revision:  msg.Document.Revision,
		Content:   msg.Document.Content,
		UpdatedAt: msg.Document.UpdatedAt,
		Resource: flare.Resource{
			ID: msg.Resource.ID,
			Change: flare.ResourceChange{
				Kind:       msg.Resource.Kind,
				DateFormat: msg.Resource.DateFormat,
			},
		},
	}, nil
}

// Init initialize the worker.
func (w *Worker) Init(options ...func(*Worker)) error {
	for _, option := range options {
		option(w)
	}

	if w.pusher == nil {
		return errors.New("pusher not found")
	}

	if w.documentRepository == nil {
		return errors.New("documentRepository not found")
	}

	if w.subscriptionTrigger == nil {
		return errors.New("subscriptionTrigger not found")
	}

	return nil
}

// WorkerPusher set the task.Pusher at Worker.
func WorkerPusher(pusher task.Pusher) func(*Worker) {
	return func(w *Worker) { w.pusher = pusher }
}

// WorkerDocumentRepository set the flare.DocumentRepositorier at Worker.
func WorkerDocumentRepository(repo flare.DocumentRepositorier) func(*Worker) {
	return func(w *Worker) { w.documentRepository = repo }
}

// WorkerSubscriptionTrigger set the subscription trigger processor.
func WorkerSubscriptionTrigger(trigger flare.SubscriptionTrigger) func(*Worker) {
	return func(w *Worker) { w.subscriptionTrigger = trigger }
}
