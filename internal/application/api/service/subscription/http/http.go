package http

import (
	"context"
	"encoding/json"
	coreHTTP "net/http"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
)

type service interface {
	Find(
		ctx context.Context, pagination infraHTTP.Pagination,
	) ([]internal.Subscription, infraHTTP.Pagination, error)
	FindByID(ctx context.Context, subscriptionID string) (*internal.Subscription, error)
	Create(ctx context.Context, subscription internal.Subscription) (string, error)
	Delete(ctx context.Context, subscriptionID string) error
	FindResource(ctx context.Context, resourceID string) (*internal.Resource, error)
}

type serviceError interface {
	error
	Client() bool
	Server() bool
	NotFound() bool
	AlreadyExists() bool
}

type subscription internal.Subscription

type response struct {
	Pagination    infraHTTP.Pagination
	Subscriptions []subscription
	Subscription  *subscription
}

func (r response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Subscription != nil {
		result = r.Subscription
	} else {
		result = map[string]interface{}{
			"pagination":    r.Pagination,
			"subscriptions": r.Subscriptions,
		}
	}

	return json.Marshal(result)
}

type subscriptionCreateEndpoint struct {
	URL     string                                `json:"url"`
	Method  string                                `json:"method"`
	Headers coreHTTP.Header                       `json:"headers"`
	Action  map[string]subscriptionCreateEndpoint `json:"actions"`
}

type subscriptionCreate struct {
	Endpoint subscriptionCreateEndpoint `json:"endpoint"`
	Delivery struct {
		Success []int `json:"success"`
		Discard []int `json:"discard"`
	} `json:"delivery"`
	Data    map[string]interface{} `json:"data"`
	Content struct {
		Document *bool `json:"document"`
		Envelope *bool `json:"envelope"`
	} `json:"content"`
}

func (s subscriptionCreate) init() error {
	// inicializar as urls
	return nil
}

func transformSubscriptions(s []internal.Subscription) []subscription {
	result := make([]subscription, len(s))
	for i := 0; i < len(s); i++ {
		result[i] = (subscription)(s[i])
	}
	return result
}

func transformResource(s *internal.Subscription) *subscription { return (*subscription)(s) }
