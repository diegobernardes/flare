// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
)

// Subscription implements flare.SubscriptionRepositorier.
type Subscription struct {
	err                error
	hasSubscriptionErr error
	triggerErr         error
	base               flare.SubscriptionRepositorier
	date               time.Time
	createId           string
}

// FindAll mock flare.SubscriptionRepositorier.FindAll.
func (r *Subscription) FindAll(
	ctx context.Context, pagination *flare.Pagination, resourceId string,
) ([]flare.Subscription, *flare.Pagination, error) {
	if r.err != nil {
		return nil, nil, r.err
	}

	subscriptions, page, err := r.base.FindAll(ctx, pagination, resourceId)
	if err != nil {
		return nil, nil, err
	}

	for i := range subscriptions {
		subscriptions[i].CreatedAt = r.date
	}

	return subscriptions, page, nil
}

// FindOne mock flare.SubscriptionRepositorier.FindOne.
func (r *Subscription) FindOne(
	ctx context.Context, resourceId, id string,
) (*flare.Subscription, error) {
	if r.err != nil {
		return nil, r.err
	}

	res, err := r.base.FindOne(ctx, resourceId, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = r.date

	return res, nil
}

// Create mock flare.SubscriptionRepositorier.Create.
func (r *Subscription) Create(ctx context.Context, subcr *flare.Subscription) error {
	if r.err != nil {
		return r.err
	}
	if r.createId != "" {
		subcr.ID = r.createId
	}

	if err := r.base.Create(ctx, subcr); err != nil {
		return err
	}
	subcr.CreatedAt = r.date
	return nil
}

// Delete mock flare.SubscriptionRepositorier.Delete.
func (r *Subscription) Delete(ctx context.Context, resourceId, id string) error {
	if r.err != nil {
		return r.err
	}
	return r.base.Delete(ctx, resourceId, id)
}

// Trigger mock flare.SubscriptionRepositorier.Trigger.
func (r *Subscription) Trigger(
	ctx context.Context,
	action string,
	document *flare.Document,
	fn func(context.Context, flare.Subscription, string) error,
) error {
	if r.triggerErr != nil {
		return r.triggerErr
	} else if r.err != nil {
		return r.err
	}
	return r.base.Trigger(ctx, action, document, fn)
}

// NewSubscription return a flare.SubscriptionRepositorier mock.
func NewSubscription(options ...func(*Subscription)) *Subscription {
	s := &Subscription{base: memory.NewSubscription()}

	for _, option := range options {
		option(s)
	}

	return s
}

// SubscriptionCreateId set id on subscription.
func SubscriptionCreateId(id string) func(*Subscription) {
	return func(s *Subscription) { s.createId = id }
}

// SubscriptionError set the error to be returned during calls.
func SubscriptionError(err error) func(*Subscription) {
	return func(s *Subscription) { s.err = err }
}

// SubscriptionTriggerError set the error to be returned during trigger calls.
func SubscriptionTriggerError(err error) func(*Subscription) {
	return func(s *Subscription) { s.triggerErr = err }
}

// SubscriptionHasSubscriptionError set the error to be returned during hasSubscription calls.
func SubscriptionHasSubscriptionError(err error) func(*Subscription) {
	return func(s *Subscription) { s.hasSubscriptionErr = err }
}

// SubscriptionDate set the date to be used at time fields.
func SubscriptionDate(date time.Time) func(*Subscription) {
	return func(s *Subscription) { s.date = date }
}

// SubscriptionLoadSliceByteSubscription load a list of encoded subscriptions in a []byte json
// layout into repository.
func SubscriptionLoadSliceByteSubscription(content []byte) func(*Subscription) {
	return func(s *Subscription) {
		subscriptions := make([]struct {
			Id       string `json:"id"`
			Endpoint struct {
				URL     string      `json:"url"`
				Method  string      `json:"method"`
				Headers http.Header `json:"headers"`
			} `json:"endpoint"`
			Delivery struct {
				Success []int `json:"success"`
				Discard []int `json:"discard"`
			} `json:"delivery"`
			Resource struct {
				Id string `json:"id"`
			} `json:"resource"`
			CreatedAt time.Time `json:"createdAt"`
		}, 0)
		if err := json.Unmarshal(content, &subscriptions); err != nil {
			panic(errors.Wrap(err,
				fmt.Sprintf("error during unmarshal of '%s' into '%v'", string(content), subscriptions),
			))
		}

		for _, rawSubscription := range subscriptions {
			uriParsed, err := url.Parse(rawSubscription.Endpoint.URL)
			if err != nil {
				panic(
					errors.Wrap(err, fmt.Sprintf("error during parse '%s' to URL", rawSubscription.Endpoint.URL)),
				)
			}

			err = s.Create(context.Background(), &flare.Subscription{
				ID:        rawSubscription.Id,
				CreatedAt: rawSubscription.CreatedAt,
				Resource:  flare.Resource{ID: rawSubscription.Resource.Id},
				Delivery: flare.SubscriptionDelivery{
					Discard: rawSubscription.Delivery.Discard,
					Success: rawSubscription.Delivery.Success,
				},
				Endpoint: flare.SubscriptionEndpoint{
					URL:     *uriParsed,
					Method:  rawSubscription.Endpoint.Method,
					Headers: rawSubscription.Endpoint.Headers,
				},
			})
			if err != nil {
				panic(errors.Wrap(err, "error during flare.Subscription persistence"))
			}
		}
	}
}
