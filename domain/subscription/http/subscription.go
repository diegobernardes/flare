// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/wildcard"
)

type pagination flare.Pagination

func (p *pagination) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}{
		Limit:  p.Limit,
		Total:  p.Total,
		Offset: p.Offset,
	})
}

type response struct {
	Pagination    *pagination
	Subscriptions []subscription
	Subscription  *subscription
}

func (r *response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Subscription != nil {
		result = r.Subscription
	} else {
		result = map[string]interface{}{"pagination": r.Pagination, "subscriptions": r.Subscriptions}
	}

	return json.Marshal(result)
}

type subscription flare.Subscription

func (s *subscription) MarshalJSON() ([]byte, error) {
	endpoint, err := s.endpointMarshalJSON(s.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "error during endpoint marshal")
	}

	actions := make(map[string]interface{})
	for action, actionEndpoint := range s.Endpoint.Action {
		content, err := s.endpointMarshalJSON(actionEndpoint)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("error during endpoint.%s marshal", action))
		}
		actions[action] = content
	}

	if len(actions) > 0 {
		endpoint["actions"] = actions
	}

	delivery := map[string][]int{
		"success": s.Delivery.Success,
		"discard": s.Delivery.Discard,
	}

	return json.Marshal(&struct {
		Id        string                 `json:"id"`
		Endpoint  map[string]interface{} `json:"endpoint"`
		Delivery  map[string][]int       `json:"delivery"`
		Content   subscriptionContent    `json:"content"`
		Data      map[string]interface{} `json:"data,omitempty"`
		CreatedAt string                 `json:"createdAt"`
	}{
		Id:        s.ID,
		Endpoint:  endpoint,
		Delivery:  delivery,
		Content:   subscriptionContent(s.Content),
		Data:      s.Data,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	})
}

func (*subscription) endpointMarshalJSON(
	endpoint flare.SubscriptionEndpoint,
) (map[string]interface{}, error) {
	content := map[string]interface{}{}

	if endpoint.URL != nil {
		endpointURL, err := url.QueryUnescape(endpoint.URL.String())
		if err != nil {
			return nil, errors.Wrap(err, "error during endpoint.url unescape")
		}
		content["url"] = endpointURL
	}

	if endpoint.Method != "" {
		content["method"] = endpoint.Method
	}

	if len(endpoint.Headers) > 0 {
		content["headers"] = endpoint.Headers
	}

	return content, nil
}

type subscriptionContent flare.SubscriptionContent

func (sc *subscriptionContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Document bool `json:"document"`
		Envelope bool `json:"envelope"`
	}{
		Document: sc.Document,
		Envelope: sc.Envelope,
	})
}

func transformSubscription(s *flare.Subscription) *subscription { return (*subscription)(s) }

func transformPagination(p *flare.Pagination) *pagination { return (*pagination)(p) }

func transformSubscriptions(s []flare.Subscription) []subscription {
	result := make([]subscription, len(s))
	for i := 0; i < len(s); i++ {
		result[i] = (subscription)(s[i])
	}
	return result
}

type subscriptionCreateEndpoint struct {
	URL     string                                `json:"url"`
	Method  string                                `json:"method"`
	Headers http.Header                           `json:"headers"`
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

func (s *subscriptionCreate) valid(resource *flare.Resource) error {
	if err := s.validEndpointURL(resource); err != nil {
		return errors.Wrap(err, "invalid endpoint")
	}

	if err := s.validEndpointMethod(resource); err != nil {
		return errors.Wrap(err, "invalid method")
	}

	if len(s.Delivery.Success) == 0 {
		return errors.New("missing delivery.Success")
	}

	if len(s.Delivery.Discard) == 0 {
		return errors.New("missing delivery.Discard")
	}

	if s.Data != nil && s.Content.Envelope != nil && !*s.Content.Envelope {
		return errors.New("could not have data while content.envelope is false")
	}

	if err := s.validData(resource); err != nil {
		return err
	}

	return nil
}

func (s *subscriptionCreate) normalize() {
	if s.Endpoint.URL != "" {
		s.Endpoint.URL = wildcard.Normalize(s.Endpoint.URL)
	}

	for action, endpoint := range s.Endpoint.Action {
		if endpoint.URL != "" {
			endpoint.URL = wildcard.Normalize(endpoint.URL)
			s.Endpoint.Action[action] = endpoint
		}
	}
}

func (s *subscriptionCreate) validEndpointURL(resource *flare.Resource) error {
	var missingURL string
	for key, action := range s.Endpoint.Action {
		if action.URL == "" {
			missingURL = key
			break
		}
	}

	if s.Endpoint.URL == "" && missingURL != "" {
		return fmt.Errorf(
			"'endpoint.url' not found while the 'endpoint.actions.%s.url' is not present", missingURL,
		)
	}

	if err := s.validEndpointURLWildcard(resource, s.Endpoint.URL, ""); err != nil {
		return err
	}

	for action, endpoint := range s.Endpoint.Action {
		if err := s.validEndpointURLWildcard(resource, endpoint.URL, action); err != nil {
			return err
		}
	}

	return nil
}

func (s *subscriptionCreate) validEndpointURLWildcard(
	resource *flare.Resource, endpoint, action string,
) error {
	if endpoint == "" {
		return nil
	}

	if err := wildcard.Valid(endpoint); err != nil {
		return errors.Wrap(err, "invalid wildcard")
	}

	resourceWildcards := wildcard.Extract(resource.Path)
	resourceWildcards = append(resourceWildcards, wildcard.Reserved...)

	endpointWildcards := wildcard.Extract(endpoint)
	if len(endpointWildcards) == 0 {
		return nil
	}

outer:
	for _, wildcard := range endpointWildcards {
		for _, rw := range resourceWildcards {
			if wildcard == rw {
				continue outer
			}
		}

		var msg string
		if action == "" {
			msg = fmt.Sprintf("endpoint.url has a wildcard '%s' that is not at the resource", wildcard)

		} else {
			msg = fmt.Sprintf(
				"endpoint.%s.url has a wildcard '%s' that is not at the resource", action, wildcard,
			)
		}

		return fmt.Errorf(msg)
	}

	return nil
}

func (s *subscriptionCreate) validEndpointMethod(resource *flare.Resource) error {
	valid := func(method string) bool {
		switch method {
		case "", http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return false
		}
		return true
	}

	var missingMethod string
	for key, action := range s.Endpoint.Action {
		if action.Method == "" {
			missingMethod = key
		}

		if !valid(action.Method) {
			return fmt.Errorf("invalid endpoint.%s.Method '%s'", key, s.Endpoint.Method)
		}
	}

	if !valid(s.Endpoint.Method) {
		return fmt.Errorf("invalid endpoint.Method '%s'", s.Endpoint.Method)
	}

	if s.Endpoint.Method == "" && missingMethod != "" {
		return fmt.Errorf(
			"'endpoint.method' not found while the 'endpoint.actions.%s.Method' is not present",
			missingMethod,
		)
	}

	return nil
}

func (s *subscriptionCreate) validData(resource *flare.Resource) error {
	for key, value := range s.Data {
		switch v := value.(type) {
		case bool, float64, string:
		case []interface{}:
			for _, content := range v {
				switch content.(type) {
				case bool, float64, string:
				default:
					return fmt.Errorf("invalid data content at key '%s'", key)
				}
			}
		default:
			return fmt.Errorf("invalid data content at key '%s'", key)
		}
	}

	return s.validDataWildcard(resource)
}

func (s *subscriptionCreate) validDataWildcard(resource *flare.Resource) error {
	resourceWildcards := wildcard.Extract(resource.Path)
	resourceWildcards = append(resourceWildcards, wildcard.Reserved...)

	for key, rawData := range s.Data {
		data, ok := rawData.(string)
		if !ok {
			continue
		}

	outer:
		for _, wildcard := range wildcard.Extract(data) {
			for _, rw := range resourceWildcards {
				if wildcard == rw {
					continue outer
				}
			}

			return fmt.Errorf("data.'%s' has a wildcard '%s' that is not at the resource", key, wildcard)
		}
	}
	return nil
}

func (s *subscriptionCreate) toFlareSubscription() (*flare.Subscription, error) {
	endpoint, err := s.toFlareSubscriptionEndpoint()
	if err != nil {
		return nil, err
	}

	subscription := &flare.Subscription{
		ID:       uuid.NewV4().String(),
		Endpoint: *endpoint,
		Delivery: flare.SubscriptionDelivery{
			Discard: s.Delivery.Discard,
			Success: s.Delivery.Success,
		},
		Data: s.Data,
	}

	if s.Content.Document != nil {
		subscription.Content.Document = *s.Content.Document
	} else {
		subscription.Content.Document = true
	}

	if s.Content.Envelope != nil {
		subscription.Content.Envelope = *s.Content.Envelope
	} else {
		subscription.Content.Envelope = true
	}

	return subscription, nil
}

func (s *subscriptionCreate) unescape() error {
	endpoint, err := url.QueryUnescape(s.Endpoint.URL)
	if err != nil {
		return errors.Wrap(err, "error during endpoint.url unescape")
	}
	s.Endpoint.URL = endpoint

	return nil
}

func (s *subscriptionCreate) toFlareSubscriptionEndpoint() (*flare.SubscriptionEndpoint, error) {
	result := flare.SubscriptionEndpoint{
		Method:  s.Endpoint.Method,
		Headers: s.Endpoint.Headers,
		Action:  make(map[string]flare.SubscriptionEndpoint),
	}

	if s.Endpoint.URL != "" {
		addr, err := url.Parse(s.Endpoint.URL)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf(
				"error during parse endpoint.url '%s' to url.URL", s.Endpoint.URL),
			)
		}
		result.URL = addr
	}

	for action, endpoint := range s.Endpoint.Action {
		ea := flare.SubscriptionEndpoint{
			Method:  endpoint.Method,
			Headers: endpoint.Headers,
		}

		if endpoint.URL != "" {
			addr, err := url.Parse(endpoint.URL)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf(
					"error during parse endpoint.%s.url '%s' to url.URL", action, s.Endpoint.URL),
				)
			}
			ea.URL = addr
		}

		result.Action[action] = ea
	}

	return &result, nil
}
