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

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraURL "github.com/diegobernardes/flare/infra/url"
	"github.com/diegobernardes/flare/infra/wildcard"
	"github.com/diegobernardes/flare/infra/worker"
)

// Delivery do the heavy lifting by discovering if the given subscription need or not to receive the
// document.
type Delivery struct {
	pusher                 worker.Pusher
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	httpClient             *http.Client
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

	err = d.subscriptionRepository.Trigger(ctx, action, document, subscription, d.trigger)
	if err != nil {
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

	if d.resourceRepository == nil {
		return errors.New("resource repository not found")
	}

	if d.subscriptionRepository == nil {
		return errors.New("subscription repository not found")
	}

	if d.httpClient == nil {
		return errors.New("httpClient not found")
	}

	return nil
}

func (d *Delivery) marshal(
	subscription *flare.Subscription, document *flare.Document, action string,
) ([]byte, error) {
	id, err := infraURL.String(document.ID)
	if err != nil {
		return nil, errors.Wrap(err, "error during document.ID unmarshal")
	}

	rawContent := struct {
		Action         string `json:"action"`
		DocumentID     string `json:"documentID"`
		ResourceID     string `json:"resourceID"`
		SubscriptionID string `json:"subscriptionID"`
	}{
		Action:         action,
		DocumentID:     id,
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

	id, err := url.Parse(value.DocumentID)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "error during parse documentID")
	}

	return &flare.Subscription{ID: value.SubscriptionID},
		&flare.Document{ID: *id, Resource: flare.Resource{ID: value.ResourceID}},
		value.Action,
		nil
}

func (d *Delivery) buildContent(
	resource *flare.Resource,
	document *flare.Document,
	documentEndpoint *url.URL,
	sub flare.Subscription,
	kind string,
) ([]byte, error) {
	var content map[string]interface{}

	if !sub.Content.Envelope {
		content = document.Content
	} else {
		id, err := infraURL.String(document.ID)
		if err != nil {
			return nil, errors.Wrap(err, "error during document.ID unmarshal")
		}

		content = map[string]interface{}{
			"id":        id,
			"action":    kind,
			"updatedAt": document.UpdatedAt.String(),
		}
		if len(sub.Data) > 0 {
			values := wildcard.ExtractValue(resource.Endpoint.Path, documentEndpoint.Path)

			for key, rawValue := range sub.Data {
				value, ok := rawValue.(string)
				if !ok {
					continue
				}
				sub.Data[key] = wildcard.Replace(value, values)
			}

			content["data"] = sub.Data
		}

		if sub.Content.Document {
			content["document"] = document.Content
		}
	}

	result, err := json.Marshal(content)
	if err != nil {
		return nil, errors.Wrap(err, "error during response generate")
	}
	return result, nil
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
	resource, err := d.resourceRepository.FindByID(ctx, document.Resource.ID)
	if err != nil {
		return nil, err
	}

	content, err := d.buildContent(resource, document, &document.ID, *subscription, action)
	if err != nil {
		return nil, errors.Wrap(err, "error during content build")
	}

	rawAddr := subscription.Endpoint.URL
	headers := subscription.Endpoint.Headers
	method := subscription.Endpoint.Method
	endpointAction, ok := subscription.Endpoint.Action[action]
	if ok {
		if endpointAction.Method != "" {
			method = endpointAction.Method
		}

		if len(endpointAction.Headers) > 0 {
			headers = endpointAction.Headers
		}

		if endpointAction.URL != nil {
			rawAddr = endpointAction.URL
		}
	}

	addr, err := d.buildEndpoint(resource, &document.ID, rawAddr)
	if err != nil {
		return nil, errors.Wrap(err, "error during endpoint generate")
	}

	req, err := d.buildRequestHTTP(ctx, content, addr, method, headers)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (d *Delivery) buildRequestHTTP(
	ctx context.Context,
	content []byte,
	addr, method string,
	headers http.Header,
) (*http.Request, error) {
	buf := bytes.NewBuffer(content)
	req, err := http.NewRequest(method, addr, buf)
	if err != nil {
		return nil, errors.Wrap(err, "error during http request create")
	}
	req = req.WithContext(ctx)

	req.Header = headers

	if req.Header == nil {
		req.Header = make(http.Header)
	}

	contentType := req.Header.Get("content-type")
	if contentType == "" && len(content) > 0 {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func (d *Delivery) buildEndpoint(
	resource *flare.Resource,
	endpoint *url.URL,
	rawSubscriptionEndpoint fmt.Stringer,
) (string, error) {
	values := wildcard.ExtractValue(resource.Endpoint.Path, endpoint.Path)
	subscriptionEndpoint, err := url.QueryUnescape(rawSubscriptionEndpoint.String())
	if err != nil {
		return "", errors.Wrap(err, "error during subscription endpoint unescape")
	}

	return wildcard.Replace(subscriptionEndpoint, values), nil
}

// DeliveryResourceRepository set the resource repository.
func DeliveryResourceRepository(repository flare.ResourceRepositorier) func(*Delivery) {
	return func(d *Delivery) { d.resourceRepository = repository }
}

// DeliverySubscriptionRepository set the subscription repository.
func DeliverySubscriptionRepository(repository flare.SubscriptionRepositorier) func(*Delivery) {
	return func(d *Delivery) { d.subscriptionRepository = repository }
}

// DeliveryPusher set the output of the messages.
func DeliveryPusher(pusher worker.Pusher) func(*Delivery) {
	return func(d *Delivery) { d.pusher = pusher }
}

// DeliveryHTTPClient set the default HTTP client to send the document changes.
func DeliveryHTTPClient(client *http.Client) func(*Delivery) {
	return func(d *Delivery) { d.httpClient = client }
}
