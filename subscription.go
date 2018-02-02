// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Subscription has all the information needed to notify the clients from changes on documents.
type Subscription struct {
	ID           string
	Endpoint     SubscriptionEndpoint
	Delivery     SubscriptionDelivery
	Resource     Resource
	Partition    string
	Data         map[string]interface{}
	SendDocument bool
	SkipEnvelope bool
	CreatedAt    time.Time
}

// SubscriptionEndpoint has the address information to notify the clients.
type SubscriptionEndpoint struct {
	URL     url.URL
	Method  string
	Headers http.Header
}

// SubscriptionDelivery is used to control whenever the notification can be considered successful
// or not.
type SubscriptionDelivery struct {
	Success []int
	Discard []int
}

// All kinds of actions a subscription trigger has.
const (
	SubscriptionTriggerCreate = "create"
	SubscriptionTriggerUpdate = "update"
	SubscriptionTriggerDelete = "delete"
)

// SubscriptionRepositorier is used to interact with the subscription data storage.
type SubscriptionRepositorier interface {
	Find(context.Context, *Pagination, string) ([]Subscription, *Pagination, error)
	FindByID(ctx context.Context, resourceID, id string) (*Subscription, error)
	FindByPartition(
		ctx context.Context, resourceID, partition string,
	) (<-chan Subscription, <-chan error, error)
	Create(context.Context, *Subscription) error
	Delete(ctx context.Context, resourceID, id string) error
	Trigger(
		ctx context.Context,
		action string,
		document *Document,
		subscription *Subscription,
		fn func(context.Context, *Document, *Subscription, string) error,
	) error
}

// SubscriptionTrigger is used to trigger the change on documents.
type SubscriptionTrigger interface {
	Push(ctx context.Context, document *Document, action string) error
}

// SubscriptionRepositoryError represents all the errors the repository can return.
type SubscriptionRepositoryError interface {
	error
	NotFound() bool
	AlreadyExists() bool
}
