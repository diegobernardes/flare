package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/diegobernardes/flare"
)

// Subscription implements the data layer for the subscription service.
type Subscription struct {
	mutex         sync.RWMutex
	subscriptions map[string][]flare.Subscription
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
		if subscription.Id == id {
			return &subscription, nil
		}
	}
	return nil, nf
}

// Create a subscription.
func (s *Subscription) Create(_ context.Context, subscription *flare.Subscription) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions, ok := s.subscriptions[subscription.Resource.Id]
	if !ok {
		s.subscriptions[subscription.Resource.Id] = make([]flare.Subscription, 0)
	}

	for _, subs := range subscriptions {
		if subs.Endpoint.URL.String() == subscription.Endpoint.URL.String() {
			return &errMemory{
				alreadyExists: true,
				message: fmt.Sprintf(
					"already exists a subscription '%s' with the endpoint.URL '%s'",
					subscription.Id,
					subscription.Endpoint.URL.String(),
				),
			}
		}
	}

	subscription.CreatedAt = time.Now()
	s.subscriptions[subscription.Resource.Id] = append(subscriptions, *subscription)
	return nil
}

// Delete a given subscription.
func (s *Subscription) Delete(_ context.Context, resourceId, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	subscriptions := s.subscriptions[resourceId]
	for i, subscription := range subscriptions {
		if subscription.Id == id {
			s.subscriptions[resourceId] = append(subscriptions[:i], subscriptions[i+1:]...)
			return nil
		}
	}

	return &errMemory{
		message:  fmt.Sprintf("subscription '%s' at resource '%s', not found", id, resourceId),
		notFound: true,
	}
}

// NewSubscription returns a configured subscription repository.
func NewSubscription() *Subscription {
	return &Subscription{subscriptions: make(map[string][]flare.Subscription)}
}
