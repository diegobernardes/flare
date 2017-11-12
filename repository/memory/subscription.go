// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/diegobernardes/flare"
)

// Subscription implements the data layer for the subscription service.
type Subscription struct {
	mutex         sync.RWMutex
	subscriptions map[string][]flare.Subscription
	changes       map[string]map[string]flare.Document
}

// FindAll returns a list of subscriptions.
func (s *Subscription) FindAll(
	_ context.Context, pagination *flare.Pagination, id string,
) ([]flare.Subscription, *flare.Pagination, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	subscriptions, ok := s.subscriptions[id]
	if !ok {
		return []flare.Subscription{}, &flare.Pagination{
			Total:  0,
			Limit:  pagination.Limit,
			Offset: pagination.Offset,
		}, nil
	}

	var resp []flare.Subscription
	if pagination.Offset > len(subscriptions) {
		resp = subscriptions
	} else if pagination.Limit+pagination.Offset > len(subscriptions) {
		resp = subscriptions[pagination.Offset:]
	} else {
		resp = subscriptions[pagination.Offset : pagination.Offset+pagination.Limit]
	}

	return resp, &flare.Pagination{
		Total:  len(subscriptions),
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}, nil
}

// FindOne return the Subscription that match the id.
func (s *Subscription) FindOne(
	_ context.Context, resourceId, id string,
) (*flare.Subscription, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	nf := &errMemory{
		message:  fmt.Sprintf("subscription '%s' at resource '%s', not found", id, resourceId),
		notFound: true,
	}
	subscriptions, ok := s.subscriptions[resourceId]
	if !ok {
		return nil, nf
	}

	for _, subscription := range subscriptions {
		if subscription.ID == id {
			return &subscription, nil
		}
	}
	return nil, nf
}

// Create a subscription.
func (s *Subscription) Create(_ context.Context, subscription *flare.Subscription) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions, ok := s.subscriptions[subscription.Resource.ID]
	if !ok {
		s.subscriptions[subscription.Resource.ID] = make([]flare.Subscription, 0)
	}

	for _, subs := range subscriptions {
		if subs.Endpoint.URL.String() == subscription.Endpoint.URL.String() {
			return &errMemory{
				alreadyExists: true,
				message: fmt.Sprintf(
					"already exists a subscription '%s' with the endpoint.URL '%s'",
					subscription.ID,
					subscription.Endpoint.URL.String(),
				),
			}
		}
	}

	subscription.CreatedAt = time.Now()
	s.subscriptions[subscription.Resource.ID] = append(subscriptions, *subscription)
	return nil
}

// HasSubscription check if a resource has subscriptions.
func (s *Subscription) HasSubscription(ctx context.Context, resourceId string) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions, ok := s.subscriptions[resourceId]
	if !ok {
		return false, nil
	}
	return len(subscriptions) > 0, nil
}

// Delete a given subscription.
func (s *Subscription) Delete(_ context.Context, resourceId, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions := s.subscriptions[resourceId]
	for i, subscription := range subscriptions {
		if subscription.ID == id {
			s.subscriptions[resourceId] = append(subscriptions[:i], subscriptions[i+1:]...)
			return nil
		}
	}

	return &errMemory{
		message:  fmt.Sprintf("subscription '%s' at resource '%s', not found", id, resourceId),
		notFound: true,
	}
}

// Trigger process the update on a document.
func (s *Subscription) Trigger(
	ctx context.Context,
	kind string,
	doc *flare.Document,
	fn func(context.Context, flare.Subscription, string) error,
) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions, ok := s.subscriptions[doc.Resource.ID]
	if !ok {
		subscriptions = make([]flare.Subscription, 0)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	for i := range subscriptions {
		group.Go(s.triggerProcess(groupCtx, subscriptions[i], doc, kind, fn))
	}

	return errors.Wrap(group.Wait(), "error during processing")
}

func (s *Subscription) triggerProcess(
	groupCtx context.Context,
	subs flare.Subscription,
	doc *flare.Document,
	kind string,

	fn func(context.Context, flare.Subscription, string) error,
) func() error {
	return func() error {
		documents, ok := s.changes[subs.ID]
		if !ok {
			documents = make(map[string]flare.Document)
			s.changes[subs.ID] = documents
		}

		referenceDocument, ok := documents[doc.Id]
		if !ok {
			if kind == flare.SubscriptionTriggerDelete {
				return nil
			}

			documents[doc.Id] = *doc
			return errors.Wrap(
				fn(groupCtx, subs, flare.SubscriptionTriggerCreate),
				"error during document subscription processing",
			)
		}

		if kind == flare.SubscriptionTriggerDelete {
			delete(documents, doc.Id)
			if err := fn(groupCtx, subs, flare.SubscriptionTriggerDelete); err != nil {
				return errors.Wrap(err, "error during document subscription processing")
			}
			return nil
		}

		newer, err := doc.Newer(&referenceDocument)
		if err != nil {
			return errors.Wrap(err, "error during check if document is newer")
		}
		if !newer {
			return nil
		}

		documents[doc.Id] = *doc
		if err := fn(groupCtx, subs, flare.SubscriptionTriggerUpdate); err != nil {
			return errors.Wrap(err, "error during document subscription processing")
		}

		return nil
	}
}

// NewSubscription returns a configured subscription repository.
func NewSubscription() *Subscription {
	return &Subscription{
		subscriptions: make(map[string][]flare.Subscription),
		changes:       make(map[string]map[string]flare.Document),
	}
}
