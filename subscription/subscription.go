// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
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
	Error         *responseError
	Subscriptions []subscription
	Subscription  *subscription
}

func (r *response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Error != nil {
		result = map[string]*responseError{"error": r.Error}
	} else if r.Subscription != nil {
		result = r.Subscription
	} else {
		result = map[string]interface{}{"pagination": r.Pagination, "subscriptions": r.Subscriptions}
	}

	return json.Marshal(result)
}

type subscription flare.Subscription

func (s *subscription) MarshalJSON() ([]byte, error) {
	endpoint := map[string]interface{}{
		"url":    s.Endpoint.URL.String(),
		"method": s.Endpoint.Method,
	}

	if len(s.Endpoint.Headers) > 0 {
		endpoint["headers"] = s.Endpoint.Headers
	}

	delivery := map[string][]int{
		"success": s.Delivery.Success,
		"discard": s.Delivery.Discard,
	}

	return json.Marshal(&struct {
		Id        string                 `json:"id"`
		Endpoint  map[string]interface{} `json:"endpoint"`
		Delivery  map[string][]int       `json:"delivery"`
		CreatedAt string                 `json:"createdAt"`
	}{
		Id:        s.Id,
		Endpoint:  endpoint,
		Delivery:  delivery,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	})
}

type responseError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
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

type subscriptionCreate struct {
	Endpoint struct {
		URL     string      `json:"url"`
		Method  string      `json:"method"`
		Headers http.Header `json:"headers"`
	} `json:"endpoint"`
	Delivery struct {
		Success []int `json:"success"`
		Discard []int `json:"discard"`
	} `json:"delivery"`
}

func (s *subscriptionCreate) valid() error {
	if s.Endpoint.URL == "" {
		return errors.New("missing endpoint.URL")
	}

	s.Endpoint.Method = strings.ToUpper(s.Endpoint.Method)
	switch s.Endpoint.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return fmt.Errorf("invalid endpoint.Method '%s'", s.Endpoint.Method)
	}

	if len(s.Delivery.Success) == 0 {
		return errors.New("missing delivery.Success")
	}

	if len(s.Delivery.Discard) == 0 {
		return errors.New("missing delivery.Discard")
	}

	return nil
}

func (s *subscriptionCreate) toFlareSubscription() (*flare.Subscription, error) {
	path, err := url.Parse(s.Endpoint.URL)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during parse '%s' to url.URL", s.Endpoint.URL))
	}

	return &flare.Subscription{
		Id: uuid.NewV4().String(),
		Endpoint: flare.SubscriptionEndpoint{
			URL:     *path,
			Method:  s.Endpoint.Method,
			Headers: s.Endpoint.Headers,
		},
		Delivery: flare.SubscriptionDelivery{
			Discard: s.Delivery.Discard,
			Success: s.Delivery.Success,
		},
	}, nil
}

func (s *Service) writeError(w http.ResponseWriter, err error, title string, status int) {
	resp := &response{Error: &responseError{Status: status}}

	if err != nil {
		resp.Error.Detail = err.Error()
	}

	if title != "" {
		resp.Error.Title = title
	}

	s.writeResponse(w, resp, status, nil)
}
