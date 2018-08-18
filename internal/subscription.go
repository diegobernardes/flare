package internal

import (
	"net/http"
	"net/url"
)

// When a subscription is created, the first state it receives is 'preparing', after all the
// background processing, it goes to 'active'. Also a subscription can be 'stopped' any time, in
// this case, all the deliveries stops.
const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusPreparing = "preparing"
	SubscriptionStatusStopped   = "stopped"
)

// The default mode is 'active'. But a subscription can also operate into 'passive' mode to have a
// better control of deliveries.
const (
	SubscriptionModeActive  = "active"
	SubscriptionModePassive = "passive"
)

const (
	SubscriptionRevisionAll  = "all"
	SubscriptionRevisionLast = "last"
)

// Subscription has all the information needed to notify the clients from changes on documents.
type Subscription struct {
	ID        string
	Endpoint  SubscriptionEndpoint
	Delivery  SubscriptionDelivery
	Resource  Resource
	Partition string // deveria estar aqui? acho que isso não faz parte do dominio...
	Data      map[string]interface{}
	Content   SubscriptionContent
	Mode      string
	Status    string
	Revision  string
}

// SubscriptionContent configure the content delived by the subscription.
type SubscriptionContent struct {
	Document bool
	Envelope bool
}

// SubscriptionEndpoint has the address information to notify the clients.
type SubscriptionEndpoint struct {
	URL     *url.URL
	Method  string
	Headers http.Header
	Action  map[string]SubscriptionEndpoint
}

// SubscriptionDelivery is used to control whenever the notification can be considered successful
// or not.
type SubscriptionDelivery struct {
	Success []int
	Discard []int
}
