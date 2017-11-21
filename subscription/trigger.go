// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/task"
)

// Trigger is used to process the signals on documents change.
type Trigger struct {
	document   flare.DocumentRepositorier
	repository flare.SubscriptionRepositorier
	httpClient *http.Client
	pusher     task.Pusher
}

func (t *Trigger) marshal(document *flare.Document, action string) ([]byte, error) {
	rawContent := map[string]interface{}{
		"action":                   action,
		"documentID":               document.Id,
		"resourceID":               document.Resource.ID,
		"changeKind":               document.Resource.Change.Kind,
		"updatedAt":                time.Now().Format(time.RFC3339),
		"documentChangeFieldValue": document.ChangeFieldValue,
	}

	if document.Resource.Change.Kind == flare.ResourceChangeDate {
		rawContent["changeDateFormat"] = document.Resource.Change.DateFormat
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (t *Trigger) unmarshal(rawContent []byte) (*flare.Document, string, error) {
	type content struct {
		Action           string      `json:"action"`
		DocumentID       string      `json:"documentID"`
		ResourceID       string      `json:"resourceID"`
		ChangeKind       string      `json:"changeKind"`
		ChangeKindFormat string      `json:"changeDateFormat"`
		UpdateAt         time.Time   `json:"updatedAt"`
		Revision         interface{} `json:"documentChangeFieldValue"`
	}

	var value content
	if err := json.Unmarshal(rawContent, &value); err != nil {
		panic(err)
	}

	resource := flare.Resource{
		ID: value.ResourceID,
		Change: flare.ResourceChange{
			Kind:       value.ChangeKind,
			DateFormat: value.ChangeKindFormat,
		},
	}

	document := &flare.Document{
		Id:               value.DocumentID,
		ChangeFieldValue: value.Revision,
		UpdatedAt:        value.UpdateAt,
		Resource:         resource,
	}
	if err := document.TransformRevision(); err != nil {
		return nil, "", errors.Wrap(err, "error during revison transformation")
	}

	return document, value.Action, nil
}

// Update the document change signal.
func (t *Trigger) Update(ctx context.Context, document *flare.Document) error {
	content, err := t.marshal(document, flare.SubscriptionTriggerUpdate)
	if err != nil {
		return errors.Wrap(err, "error during trigger")
	}

	if err = t.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

// Delete the document change signal.
func (t *Trigger) Delete(ctx context.Context, document *flare.Document) error {
	content, err := t.marshal(document, flare.SubscriptionTriggerDelete)
	if err != nil {
		return errors.Wrap(err, "error during trigger")
	}

	if err = t.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

// Process is used to consume the tasks.
func (t *Trigger) Process(ctx context.Context, rawContent []byte) error {
	rawDocument, action, err := t.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal the message")
	}

	document, err := t.document.FindOneWithRevision(ctx, rawDocument.Id, rawDocument.ChangeFieldValue)
	if err != nil {
		return errors.Wrap(err, "error during document find")
	}

	if err = t.repository.Trigger(ctx, action, document, t.exec(document)); err != nil {
		return errors.Wrap(err, "error during message process")
	}
	return nil
}

func (t *Trigger) exec(
	document *flare.Document,
) func(context.Context, flare.Subscription, string) error {
	return func(ctx context.Context, sub flare.Subscription, kind string) error {
		content, err := t.buildContent(document, sub, kind)
		if err != nil {
			return errors.Wrap(err, "error during content build")
		}

		buf := bytes.NewBuffer(content)
		req, err := http.NewRequest(sub.Endpoint.Method, sub.Endpoint.URL.String(), buf)
		if err != nil {
			return errors.Wrap(err, "error during http request create")
		}
		req = req.WithContext(ctx)

		for key, values := range sub.Endpoint.Headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		req.Header.Add("Content-Type", "application/json")

		resp, err := t.httpClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "error during http request")
		}

		for _, status := range sub.Delivery.Success {
			if status == resp.StatusCode {
				return nil
			}
		}

		for _, status := range sub.Delivery.Discard {
			if status == resp.StatusCode {
				return nil
			}
		}

		return errors.Errorf(
			"success and discard status don't match with the response value '%d'", resp.StatusCode,
		)
	}
}

func (t *Trigger) buildContent(
	document *flare.Document, sub flare.Subscription, kind string,
) ([]byte, error) {
	rawContent := map[string]interface{}{
		"id":        document.Id,
		"action":    kind,
		"updatedAt": document.UpdatedAt.String(),
	}
	if sub.Data != nil {
		replacer, err := sub.Resource.WildcardReplace(document.Id, document.ChangeFieldValue)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract the wildcards from document id")
		}

		for key, rawValue := range sub.Data {
			if value, ok := rawValue.(string); ok {
				sub.Data[key] = replacer(value)
			}
		}

		rawContent["data"] = sub.Data
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during response generate")
	}
	return content, nil
}

// Init initialize the Trigger.
func (t *Trigger) Init(options ...func(*Trigger)) error {
	for _, option := range options {
		option(t)
	}

	if t.document == nil {
		return errors.New("document repository not found")
	}

	if t.pusher == nil {
		return errors.New("pusher not found")
	}

	if t.repository == nil {
		return errors.New("repository not found")
	}

	if t.httpClient == nil {
		return errors.New("httpClient not found")
	}

	return nil
}

// TriggerRepository set the repository on Trigger.
func TriggerRepository(repository flare.SubscriptionRepositorier) func(*Trigger) {
	return func(t *Trigger) {
		t.repository = repository
	}
}

// TriggerHTTPClient set the httpClient on Trigger.
func TriggerHTTPClient(httpClient *http.Client) func(*Trigger) {
	return func(t *Trigger) {
		t.httpClient = httpClient
	}
}

// TriggerPusher set the pusher that gonna receive the trigger notifications.
func TriggerPusher(pusher task.Pusher) func(*Trigger) {
	return func(t *Trigger) {
		t.pusher = pusher
	}
}

// TriggerDocumentRepository set the document repository.
func TriggerDocumentRepository(repo flare.DocumentRepositorier) func(*Trigger) {
	return func(t *Trigger) {
		t.document = repo
	}
}
