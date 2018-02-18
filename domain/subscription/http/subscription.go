// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	retry := map[string]interface{}{
		"interval":    s.Delivery.Retry.Interval.String(),
		"progression": s.Delivery.Retry.Progression,
	}

	if s.Delivery.Retry.Ratio != 0 {
		retry["ratio"] = s.Delivery.Retry.Ratio
	}

	if s.Delivery.Retry.TTL != 0 {
		retry["ttl"] = s.Delivery.Retry.TTL.String()
	}

	if s.Delivery.Retry.Quantity != 0 {
		retry["quantity"] = s.Delivery.Retry.Quantity
	}

	delivery := map[string]interface{}{
		"success": s.Delivery.Success,
		"discard": s.Delivery.Discard,
		"retry":   retry,
	}

	return json.Marshal(&struct {
		Id        string                 `json:"id"`
		Endpoint  map[string]interface{} `json:"endpoint"`
		Delivery  map[string]interface{} `json:"delivery"`
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

func unmarshalSubscription(s *flare.Subscription) *subscription { return (*subscription)(s) }

func unmarshalPagination(p *flare.Pagination) *pagination { return (*pagination)(p) }

func unmarshalSubscriptions(s []flare.Subscription) []subscription {
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
		Retry   struct {
			Interval       string `json:"interval"`
			parsedInterval time.Duration

			TTL       string `json:"ttl"`
			parsedTTL time.Duration

			Quantity    int     `json:"quantity"`
			Progression string  `json:"progression"`
			Ratio       float64 `json:"ratio"`
		} `json:"retry"`
	} `json:"delivery"`
	Data    map[string]interface{} `json:"data"`
	Content struct {
		Document *bool `json:"document"`
		Envelope *bool `json:"envelope"`
	} `json:"content"`
}

func (s *subscriptionCreate) parse(body io.Reader) error {
	fn := func(content, field string) (time.Duration, error) {
		if content == "" {
			return 0, nil
		}

		duration, err := time.ParseDuration(content)
		if err != nil {
			return 0, errors.Wrap(
				err,
				fmt.Sprintf("invalid delivery.retry.%s '%s'", field, content),
			)
		}

		if duration < 0 {
			return 0, errors.Wrap(
				err,
				fmt.Sprintf("can't have negative delivery.retry.%s '%s'", field, content),
			)
		}

		return duration, nil
	}

	d := json.NewDecoder(body)

	if err := d.Decode(s); err != nil {
		return errors.Wrap(err, "error during body unmarshal")
	}

	if s.Delivery.Retry.Progression == "" {
		s.Delivery.Retry.Progression = flare.SubscriptionDeliveryRetryProgressionLinear
	}

	var err error
	s.Delivery.Retry.parsedInterval, err = fn(s.Delivery.Retry.Interval, "interval")
	if err != nil {
		return errors.Wrap(err, "error during parse delivery.retry.interval")
	}

	s.Delivery.Retry.parsedTTL, err = fn(s.Delivery.Retry.TTL, "ttl")
	if err != nil {
		return errors.Wrap(err, "error during parse delivery.retry.ttl")
	}

	if err := s.unescape(); err != nil {
		return errors.Wrap(err, "error during subscription create delivery endpoint unescape")
	}

	s.Endpoint.Method = strings.ToUpper(s.Endpoint.Method)
	for action, endpoint := range s.Endpoint.Action {
		endpoint.Method = strings.ToUpper(endpoint.Method)
		s.Endpoint.Action[action] = endpoint
	}

	s.normalize()
	return nil
}

func (s *subscriptionCreate) valid(resource *flare.Resource) error {
	if err := s.validEndpointURL(resource); err != nil {
		return errors.Wrap(err, "invalid endpoint")
	}

	if err := s.validEndpointMethod(); err != nil {
		return errors.Wrap(err, "invalid method")
	}

	if err := s.validDeliveryRetry(); err != nil {
		return errors.Wrap(err, "invalid delivery.retry")
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

func (s *subscriptionCreate) validDeliveryRetry() error {
	if s.Delivery.Retry.Quantity < 0 {
		return errors.New("delivery.retry.quantity should be bigger then zero")
	}

	if s.Delivery.Retry.parsedTTL < 0 {
		return errors.New("delivery.retry.ttl should be bigger or equal zero")
	}

	if s.Delivery.Retry.parsedInterval <= 0 {
		return errors.New("delivery.retry.interval should be bigger then zero")
	}

	progression := s.Delivery.Retry.Progression
	switch progression {
	case
		flare.SubscriptionDeliveryRetryProgressionLinear,
		flare.SubscriptionDeliveryRetryProgressionArithmetic,
		flare.SubscriptionDeliveryRetryProgressionGeometric:
	default:
		return errors.Errorf("invalid delivery.retry.progression '%s'", progression)
	}

	if progression == flare.SubscriptionDeliveryRetryProgressionLinear && s.Delivery.Retry.Ratio != 0 {
		return errors.Errorf("delivery.retry.progression.linear don't require a ratio")
	}

	if s.Delivery.Retry.Ratio != 0 && s.Delivery.Retry.Ratio < 1 {
		return errors.New("delivery.retry.ratio can't be less then 1")
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
	if err := s.validEndpointURLPresence(); err != nil {
		return err
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

func (s *subscriptionCreate) validEndpointURLPresence() error {
	actions := []string{
		flare.SubscriptionTriggerCreate,
		flare.SubscriptionTriggerUpdate,
		flare.SubscriptionTriggerDelete,
	}

	var missingURL string
	for _, action := range actions {
		endpoint, ok := s.Endpoint.Action[action]
		if !ok {
			missingURL = action
			break
		}

		if endpoint.URL == "" {
			missingURL = action
			break
		}
	}

	if s.Endpoint.URL == "" && len(s.Endpoint.Action) == 0 {
		return errors.New("'endpoint.url' not found")
	} else if s.Endpoint.URL == "" && missingURL != "" {
		return fmt.Errorf(
			"'endpoint.url' not found while the 'endpoint.actions.%s.url' is not present", missingURL,
		)
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

	resourceWildcards := wildcard.Extract(resource.Endpoint.Path)
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

func (s *subscriptionCreate) validEndpointMethod() error {
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
	resourceWildcards := wildcard.Extract(resource.Endpoint.Path)
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

func (s *subscriptionCreate) marshal() (*flare.Subscription, error) {
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
			Retry: flare.SubscriptionDeliveryRetry{
				TTL:         s.Delivery.Retry.parsedTTL,
				Quantity:    s.Delivery.Retry.Quantity,
				Interval:    s.Delivery.Retry.parsedInterval,
				Progression: s.Delivery.Retry.Progression,
				Ratio:       s.Delivery.Retry.Ratio,
			},
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

	for action, actionEndpoint := range s.Endpoint.Action {
		if actionEndpoint.URL != "" {
			endpoint, err := url.QueryUnescape(actionEndpoint.URL)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error during endpoint.action.%s.url unescape", action))
			}
			actionEndpoint.URL = endpoint
			s.Endpoint.Action[action] = actionEndpoint
		}
	}

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
