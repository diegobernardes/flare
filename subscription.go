package flare

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Subscription is used to notify the clients about changes on documents.
type Subscription struct {
	Id        string
	Endpoint  SubscriptionEndpoint
	Delivery  SubscriptionDelivery
	Resource  Resource
	CreatedAt time.Time
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

// All kinds of actions a subscription trigger supports.
const (
	SubscriptionTriggerInsert = "insert"
	SubscriptionTriggerUpdate = "update"
	SubscriptionTriggerDelete = "delete"
)

// SubscriptionRepositorier is used to interact with the Subscription data storage.
type SubscriptionRepositorier interface {
	FindAll(context.Context, *Pagination, string) ([]Subscription, *Pagination, error)
	FindOne(ctx context.Context, resourceId, id string) (*Subscription, error)
	Create(context.Context, *Subscription) error
	Delete(ctx context.Context, resourceId, id string) error
}

// SubscriptionRepositoryError implements all the errrors the repository can return.
type SubscriptionRepositoryError interface {
	NotFound() bool
	AlreadyExists() bool
}
