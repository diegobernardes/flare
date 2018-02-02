// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/worker"
)

// Delivery do the heavy lifting by discovering if the given subscription need or not to receive the
// document.
type Delivery struct {
	pusher     worker.Pusher
	repository flare.SubscriptionRepositorier
	httpClient *http.Client
}

// Push the signal to delivery the document.
func (d *Delivery) Push(
	ctx context.Context, subscription *flare.Subscription, document *flare.Document, action string,
) error {
	content, err := d.marshal(subscription, document, action)
	if err != nil {
		return errors.Wrap(err, "error during trigger")
	}

	if err = d.pusher.Push(ctx, content); err != nil {
		return errors.Wrap(err, "error during message delivery")
	}
	return nil
}

// Process the message.
func (d *Delivery) Process(ctx context.Context, rawContent []byte) error {
	subscription, document, action, err := d.unmarshal(rawContent)
	if err != nil {
		return errors.Wrap(err, "error during content unmarshal")
	}

	if err := d.repository.Trigger(ctx, action, document, subscription, d.trigger); err != nil {
		return errors.Wrap(err, "error during subscription trigger")
	}
	return nil
}

// Init initialize the Delivery.
func (d *Delivery) Init(options ...func(*Delivery)) error {
	for _, option := range options {
		option(d)
	}

	if d.pusher == nil {
		return errors.New("pusher not found")
	}

	if d.repository == nil {
		return errors.New("repository not found")
	}

	if d.httpClient == nil {
		return errors.New("httpClient not found")
	}

	return nil
}

func (d *Delivery) marshal(
	subscription *flare.Subscription, document *flare.Document, action string,
) ([]byte, error) {
	rawContent := struct {
		Action         string `json:"action"`
		DocumentID     string `json:"documentID"`
		ResourceID     string `json:"resourceID"`
		SubscriptionID string `json:"subscriptionID"`
	}{
		Action:         action,
		DocumentID:     document.ID,
		ResourceID:     document.Resource.ID,
		SubscriptionID: subscription.ID,
	}

	content, err := json.Marshal(rawContent)
	if err != nil {
		return nil, errors.Wrap(err, "error during message marshal")
	}
	return content, nil
}

func (d *Delivery) unmarshal(
	rawContent []byte,
) (*flare.Subscription, *flare.Document, string, error) {
	type content struct {
		Action         string `json:"action"`
		DocumentID     string `json:"documentID"`
		ResourceID     string `json:"resourceID"`
		SubscriptionID string `json:"subscriptionID"`
	}

	var value content
	if err := json.Unmarshal(rawContent, &value); err != nil {
		return nil, nil, "", errors.Wrap(err, "error during content unmarshal")
	}

	return &flare.Subscription{ID: value.SubscriptionID},
		&flare.Document{ID: value.DocumentID, Resource: flare.Resource{ID: value.ResourceID}},
		value.Action,
		nil
}

func (d *Delivery) buildContent(
	document *flare.Document, sub flare.Subscription, kind string,
) ([]byte, error) {
	var content map[string]interface{}

	if sub.SkipEnvelope {
		content = document.Content
	} else {
		content = map[string]interface{}{
			"id":        document.ID,
			"action":    kind,
			"updatedAt": document.UpdatedAt.String(),
		}
		if len(sub.Data) > 0 {
			replacer, err := d.wildcardReplace(&sub.Resource, document)
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract the wildcards from document id")
			}

			for key, rawValue := range sub.Data {
				if value, ok := rawValue.(string); ok {
					sub.Data[key] = replacer(value)
				}
			}

			content["data"] = sub.Data
		}

		if sub.SendDocument {
			content["document"] = document.Content
		}
	}

	result, err := json.Marshal(content)
	if err != nil {
		return nil, errors.Wrap(err, "error during response generate")
	}
	return result, nil
}

func (d *Delivery) wildcardReplace(
	r *flare.Resource, doc *flare.Document,
) (func(string) string, error) {
	endpoint, err := url.Parse(doc.ID)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url parse of '%s'", doc.ID))
	}
	wildcards := strings.Split(r.Path, "/")
	documentWildcards := strings.Split(endpoint.Path, "/")

	return func(value string) string {
		for i, wildcard := range wildcards {
			if wildcard == "" {
				continue
			}

			if wildcard[0] == '{' && wildcard[len(wildcard)-1] == '}' {
				value = strings.Replace(value, wildcard, documentWildcards[i], -1)
			}
		}
		return value
	}, nil
}

func (d *Delivery) trigger(
	ctx context.Context,
	document *flare.Document,
	subscription *flare.Subscription,
	action string,
) error {
	req, err := d.buildRequest(ctx, document, subscription, action)
	if err != nil {
		return err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "error during http request")
	}

	for _, status := range subscription.Delivery.Success {
		if status == resp.StatusCode {
			return nil
		}
	}

	for _, status := range subscription.Delivery.Discard {
		if status == resp.StatusCode {
			return nil
		}
	}

	return errors.Errorf(
		"success and discard status don't match with the response value '%d'", resp.StatusCode,
	)
}

func (d *Delivery) buildRequest(
	ctx context.Context,
	document *flare.Document,
	subscription *flare.Subscription,
	action string,
) (*http.Request, error) {
	content, err := d.buildContent(document, *subscription, action)
	if err != nil {
		return nil, errors.Wrap(err, "error during content build")
	}

	buf := bytes.NewBuffer(content)
	req, err := http.NewRequest(subscription.Endpoint.Method, subscription.Endpoint.URL.String(), buf)
	if err != nil {
		return nil, errors.Wrap(err, "error during http request create")
	}
	req = req.WithContext(ctx)

	for key, values := range subscription.Endpoint.Headers {
		for _, value := range values {
			if key == "Content-Type" && value == "application/json" && len(content) > 0 {
				continue
			}
			req.Header.Add(key, value)
		}
	}

	if len(content) > 0 {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// DeliverySubscriptionRepository set the subscription repository.
func DeliverySubscriptionRepository(repository flare.SubscriptionRepositorier) func(*Delivery) {
	return func(d *Delivery) { d.repository = repository }
}

// DeliveryPusher set the output of the messages.
func DeliveryPusher(pusher worker.Pusher) func(*Delivery) {
	return func(d *Delivery) { d.pusher = pusher }
}

// DeliveryHTTPClient set the default HTTP client to send the document changes.
func DeliveryHTTPClient(client *http.Client) func(*Delivery) {
	return func(d *Delivery) { d.httpClient = client }
}
