// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Subscription implements the data layer for the subscription service.
type Subscription struct {
	mutex              sync.RWMutex
	resourceRepository resourceRepositorier
	documentRepository flare.DocumentRepositorier

	// resourceID -> []subscription
	subscriptions map[string][]flare.Subscription

	// subscriptionID -> documentID -> document revision
	changes map[string]map[string]int64
}

// Find returns a list of subscriptions.
func (s *Subscription) Find(
	_ context.Context, pagination *flare.Pagination, resourceID string,
) ([]flare.Subscription, *flare.Pagination, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	subscriptions, ok := s.subscriptions[resourceID]
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

// FindByID return the Subscription that match the id.
func (s *Subscription) FindByID(
	ctx context.Context, resourceID, id string,
) (*flare.Subscription, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.findByID(ctx, resourceID, id)
}

func (s *Subscription) findByID(
	_ context.Context, resourceID, id string,
) (*flare.Subscription, error) {
	nf := &errMemory{
		message:  fmt.Sprintf("subscription '%s' at resource '%s', not found", id, resourceID),
		notFound: true,
	}
	subscriptions, ok := s.subscriptions[resourceID]
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

// FindByPartition find all subscriptions that belongs to a given partition.
func (s *Subscription) FindByPartition(
	_ context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	chanResult := make(chan flare.Subscription)
	chanErr := make(chan error)
	s.mutex.Lock()

	go func() {
		defer func() {
			close(chanResult)
			s.mutex.Unlock()
		}()

		subscriptions, ok := s.subscriptions[resourceID]
		if !ok {
			return
		}

		for _, subscription := range subscriptions {
			if subscription.Partition == partition {
				chanResult <- subscription
			}
		}
	}()

	return chanResult, chanErr, nil
}

// Create a subscription.
func (s *Subscription) Create(ctx context.Context, subscription *flare.Subscription) error {
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

	partition, err := s.resourceRepository.joinPartition(ctx, subscription.Resource.ID)
	if err != nil {
		return errors.Wrap(err, "error during join partition")
	}

	subscription.Partition = partition
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
func (s *Subscription) Delete(ctx context.Context, resourceId, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions := s.subscriptions[resourceId]
	for i, subscription := range subscriptions {
		if subscription.ID == id {
			err := s.resourceRepository.leavePartition(ctx, subscription.Resource.ID, subscription.Partition)
			if err != nil {
				return errors.Wrap(err, "error during leave partition")
			}

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
	action string,
	rawDocument *flare.Document,
	rawSubscription *flare.Subscription,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscription, doc, err := s.triggerDocumentAndSubscription(
		ctx, action, rawDocument, rawSubscription,
	)
	if err != nil {
		panic(err)
	}

	subscriptionMap, ok := s.changes[subscription.ID]
	if !ok {
		if action == flare.SubscriptionTriggerDelete {
			return nil
		}
		return s.triggerProcess(ctx, subscription, doc, flare.SubscriptionTriggerCreate, fn)
	}

	revision, ok := subscriptionMap[doc.ID]
	if !ok {
		if action == flare.SubscriptionTriggerDelete {
			return nil
		}
		return s.triggerProcess(ctx, subscription, doc, flare.SubscriptionTriggerCreate, fn)
	}

	reference, err := s.documentRepository.FindByID(ctx, doc.ID)
	if err != nil {
		return errors.Wrap(
			err,
			"error while loading reference document to process the suscription trigger",
		)
	}

	if action == flare.SubscriptionTriggerDelete {
		return s.triggerProcess(ctx, subscription, doc, flare.SubscriptionTriggerDelete, fn)
	}

	if reference.Revision > revision {
		return s.triggerProcess(ctx, subscription, doc, flare.SubscriptionTriggerUpdate, fn)
	}
	return nil
}

func (s *Subscription) triggerDocumentAndSubscription(
	ctx context.Context,
	action string,
	rawDocument *flare.Document,
	rawSubscription *flare.Subscription,
) (*flare.Subscription, *flare.Document, error) {
	subscription, err := s.findByID(ctx, rawDocument.Resource.ID, rawSubscription.ID)
	if err != nil {
		if repoErr := err.(flare.SubscriptionRepositoryError); repoErr.NotFound() {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	doc, err := s.documentRepository.FindByID(ctx, rawDocument.ID)
	if err != nil {
		if repoErr := err.(flare.DocumentRepositoryError); repoErr.NotFound() {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return subscription, doc, nil
}

func (s *Subscription) triggerProcess(
	ctx context.Context,
	subs *flare.Subscription,
	doc *flare.Document,
	action string,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	if err := fn(ctx, doc, subs, action); err != nil {
		return err
	}

	subscriptionMap, ok := s.changes[subs.ID]
	if !ok {
		subscriptionMap = make(map[string]int64)
		s.changes[subs.ID] = subscriptionMap
	}

	if action == flare.SubscriptionTriggerDelete {
		delete(subscriptionMap, doc.ID)
		return nil
	}

	subscriptionMap[doc.ID] = doc.Revision
	return nil
}

func (s *Subscription) init() {
	s.subscriptions = make(map[string][]flare.Subscription)
	s.changes = make(map[string]map[string]int64)
}
