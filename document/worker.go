// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/task"
)

// Worker is used to async process all the create, update and delete operations on documents.
type Worker struct {
	pusher                 task.Pusher
	resourceRepository     flare.ResourceRepositorier
	documentRepository     flare.DocumentRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	subscriptionTrigger    flare.SubscriptionTrigger
}

func (w *Worker) push(ctx context.Context, id, action string, body []byte) error {
	content, err := w.marshal(id, action, body)
	if err != nil {
		return errors.Wrap(err, "error during message compress")
	}

	if err = w.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during job enqueue")
	}
	return nil
}

// Process process the enqueued documents.
func (w *Worker) Process(ctx context.Context, rawContent []byte) error {
	content, id, action, err := w.extractContent(rawContent)
	if err != nil {
		return errors.Wrap(err, "error during message uncompress")
	}

	switch action {
	case flare.SubscriptionTriggerCreate, flare.SubscriptionTriggerUpdate:
		rawBody, ok := content["body"].(string)
		if !ok {
			return errors.New("missing body content")
		}

		body := []byte(rawBody)
		if err = w.processUpdate(ctx, id, action, body); err != nil {
			var msg string
			if action == flare.SubscriptionTriggerCreate {
				msg = "error during document create"
			} else {
				msg = "error during document update"
			}
			return errors.Wrap(err, msg)
		}
	case flare.SubscriptionTriggerDelete:
		if err = w.processDelete(ctx, id); err != nil {
			return errors.Wrap(err, "error during document delete")
		}
	default:
		return fmt.Errorf("action '%s' not supported", action)
	}

	return nil
}

func (w *Worker) extractContent(rawContent []byte) (map[string]interface{}, string, string, error) {
	content, err := w.unmarshal(rawContent)
	if err != nil {
		return nil, "", "", errors.Wrap(err, "error during message uncompress")
	}

	id, ok := content["id"].(string)
	if !ok {
		return nil, "", "", errors.New("missing id content")
	}

	action, ok := content["action"].(string)
	if !ok {
		return nil, "", "", errors.New("missing action content")
	}

	return content, id, action, nil
}

func (w *Worker) processDelete(ctx context.Context, id string) error {
	document, err := w.documentRepository.FindOne(ctx, id)
	if err != nil {
		return errors.Wrap(err, "error during the check if the document exists")
	}

	if err = w.subscriptionTrigger.Delete(ctx, document); err != nil {
		return errors.Wrap(err, "error during document change trigger")
	}

	if err = w.documentRepository.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "error during delete")
	}
	return nil
}

func (w *Worker) processUpdate(ctx context.Context, id, action string, body []byte) error {
	document, err := w.parseHandleUpdateDocument(ctx, body, id)
	if err != nil {
		return errors.Wrap(err, "could not parse the document")
	}

	referenceDocument, err := w.documentRepository.FindOne(ctx, document.Id)
	if err != nil {
		if _, ok := err.(flare.DocumentRepositoryError); !ok {
			return errors.Wrap(err, "error during document search")
		}
	}

	hasSubscr, err := w.subscriptionRepository.HasSubscription(ctx, document.Resource.ID)
	if err != nil {
		return errors.Wrap(err, "error during check if the document resource has subscriptions")
	}
	if !hasSubscr {
		return nil
	}

	if err = w.updateAndTriggerDocumentChange(ctx, document, referenceDocument, action); err != nil {
		return errors.Wrap(err, "error during document change trigger")
	}
	return nil
}

func (w *Worker) marshal(id, action string, body []byte) ([]byte, error) {
	content, err := json.Marshal(map[string]interface{}{
		"id":     id,
		"action": action,
		"body":   string(body),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (w *Worker) unmarshal(rawContent []byte) (map[string]interface{}, error) {
	content := make(map[string]interface{})
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, errors.Wrap(err, "error during message unmarshal")
	}
	return content, nil
}

func (w *Worker) parseHandleUpdateDocument(
	ctx context.Context, rawContent []byte, id string,
) (*flare.Document, error) {
	content := make(map[string]interface{})
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, errors.Wrap(err, "invalid body content")
	}

	resource, err := w.resourceRepository.FindByURI(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource search")
	}

	document := &flare.Document{
		Id:               id,
		ChangeFieldValue: content[resource.Change.Field],
		Resource:         *resource,
	}
	if err = document.Valid(); err != nil {
		return nil, errors.Wrap(err, "document is not valid")
	}

	return document, nil
}

func (w *Worker) updateAndTriggerDocumentChange(
	ctx context.Context, document, referenceDocument *flare.Document, action string,
) error {
	var (
		newer bool
		err   error
	)

	if referenceDocument != nil {
		newer, err = document.Newer(referenceDocument)
		if err != nil {
			return errors.Wrap(err, "error during comparing the document with the latest one on datastorage")
		}
	}

	if newer ||
		action == flare.SubscriptionTriggerCreate ||
		action == flare.SubscriptionTriggerUpdate {
		if err = w.documentRepository.Update(ctx, document); err != nil {
			return errors.Wrap(err, "error during document persistence")
		}
	}

	if action == flare.SubscriptionTriggerCreate || action == flare.SubscriptionTriggerUpdate {
		if err = w.subscriptionTrigger.Update(ctx, document); err != nil {
			return errors.Wrap(err, "error during document change trigger")
		}
	}
	return nil
}

// Init initialize the worker.
func (w *Worker) Init(options ...func(*Worker)) error {
	for _, option := range options {
		option(w)
	}

	if w.pusher == nil {
		return errors.New("pusher not found")
	}

	if w.resourceRepository == nil {
		return errors.New("resourceRepository not found")
	}

	if w.documentRepository == nil {
		return errors.New("documentRepository not found")
	}

	if w.subscriptionRepository == nil {
		return errors.New("subscriptionRepository not found")
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

// WorkerResourceRepository set the flare.ResourceRepositorier at Worker.
func WorkerResourceRepository(repo flare.ResourceRepositorier) func(*Worker) {
	return func(w *Worker) { w.resourceRepository = repo }
}

// WorkerDocumentRepository set the flare.DocumentRepositorier at Worker.
func WorkerDocumentRepository(repo flare.DocumentRepositorier) func(*Worker) {
	return func(w *Worker) { w.documentRepository = repo }
}

// WorkerSubscriptionRepository set the repository to access the subscriptions.
func WorkerSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Worker) {
	return func(w *Worker) { w.subscriptionRepository = repo }
}

// WorkerSubscriptionTrigger set the subscription trigger processor.
func WorkerSubscriptionTrigger(trigger flare.SubscriptionTrigger) func(*Worker) {
	return func(w *Worker) { w.subscriptionTrigger = trigger }
}
