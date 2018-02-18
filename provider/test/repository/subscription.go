// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
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

// Find mock flare.SubscriptionRepositorier.FindAll.
func (s *Subscription) Find(
	ctx context.Context, pagination *flare.Pagination, resourceId string,
) ([]flare.Subscription, *flare.Pagination, error) {
	if s.err != nil {
		return nil, nil, s.err
	}

	subscriptions, page, err := s.base.Find(ctx, pagination, resourceId)
	if err != nil {
		return nil, nil, err
	}

	for i := range subscriptions {
		subscriptions[i].CreatedAt = s.date
	}

	return subscriptions, page, nil
}

// FindByID mock flare.SubscriptionRepositorier.FindOne.
func (s *Subscription) FindByID(
	ctx context.Context, resourceId, id string,
) (*flare.Subscription, error) {
	if s.err != nil {
		return nil, s.err
	}

	res, err := s.base.FindByID(ctx, resourceId, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = s.date

	return res, nil
}

// FindByPartition mock flare.SubscriptionRepositorier.FindByPartition
func (s *Subscription) FindByPartition(
	ctx context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	if s.err != nil {
		return nil, nil, s.err
	}
	return s.base.FindByPartition(ctx, resourceID, partition)
}

// Create mock flare.SubscriptionRepositorier.Create.
func (s *Subscription) Create(ctx context.Context, subcr *flare.Subscription) error {
	if s.err != nil {
		return s.err
	}
	if s.createId != "" {
		subcr.ID = s.createId
	}

	if err := s.base.Create(ctx, subcr); err != nil {
		return err
	}
	subcr.CreatedAt = s.date
	return nil
}

// Delete mock flare.SubscriptionRepositorier.Delete.
func (s *Subscription) Delete(ctx context.Context, resourceId, id string) error {
	if s.err != nil {
		return s.err
	}
	return s.base.Delete(ctx, resourceId, id)
}

// Trigger mock flare.SubscriptionRepositorier.Trigger.
func (s *Subscription) Trigger(
	ctx context.Context,
	action string,
	document *flare.Document,
	subscription *flare.Subscription,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	if s.triggerErr != nil {
		return s.triggerErr
	} else if s.err != nil {
		return s.err
	}
	return s.base.Trigger(ctx, action, document, subscription, fn)
}

func newSubscription(options ...func(*Subscription)) *Subscription {
	s := &Subscription{}

	for _, option := range options {
		option(s)
	}

	return s
}

// SubscriptionRepository set the subscription repository.
func SubscriptionRepository(repository flare.SubscriptionRepositorier) func(*Subscription) {
	return func(s *Subscription) { s.base = repository }
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

// SubscriptionLoadSliceByteSubscription load a list of encoded subscriptions layout into
// repository.
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
				Retry   struct {
					Interval    string
					TTL         string
					Quantity    int
					Progression string
					Ratio       float64
				} `json:"retry"`
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

			retry := flare.SubscriptionDeliveryRetry{
				Progression: rawSubscription.Delivery.Retry.Progression,
				Quantity:    rawSubscription.Delivery.Retry.Quantity,
				Ratio:       rawSubscription.Delivery.Retry.Ratio,
			}

			if rawSubscription.Delivery.Retry.Interval != "" {
				retry.Interval, err = time.ParseDuration(rawSubscription.Delivery.Retry.Interval)
				if err != nil {
					panic(err)
				}
			}

			if rawSubscription.Delivery.Retry.TTL != "" {
				retry.TTL, err = time.ParseDuration(rawSubscription.Delivery.Retry.TTL)
				if err != nil {
					panic(err)
				}
			}

			err = s.Create(context.Background(), &flare.Subscription{
				ID:        rawSubscription.Id,
				CreatedAt: rawSubscription.CreatedAt,
				Resource:  flare.Resource{ID: rawSubscription.Resource.Id},
				Delivery: flare.SubscriptionDelivery{
					Discard: rawSubscription.Delivery.Discard,
					Success: rawSubscription.Delivery.Success,
					Retry:   retry,
				},
				Endpoint: flare.SubscriptionEndpoint{
					URL:     uriParsed,
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
