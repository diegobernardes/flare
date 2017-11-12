// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

func TestPaginationMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  pagination
		output string
		hasErr bool
	}{
		{
			"Valid pagination",
			pagination{Limit: 30, Offset: 0},
			`{"limit":30,"offset":0,"total":0}`,
			false,
		},
		{
			"Valid pagination",
			pagination{Limit: 10, Offset: 30, Total: 120},
			`{"limit":10,"offset":30,"total":120}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("pagination.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			if string(content) != tt.output {
				t.Errorf(
					"pagination.MarshalJSON invalid result, want '%v', got '%v'", string(content), tt.output,
				)
			}
		})
	}
}

func TestResponseMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  response
		output []byte
		hasErr bool
	}{
		{
			"Valid JSON",
			response{
				Error: &responseError{
					Status: http.StatusBadRequest,
					Title:  "error during query",
					Detail: "detail from error",
				},
			},
			load("responseMarshalJSON.valid1.json"),
			false,
		},
		{
			"Valid JSON",
			response{
				Error: &responseError{
					Status: http.StatusServiceUnavailable,
					Title:  "service unavailable",
				},
			},
			load("responseMarshalJSON.valid2.json"),
			false,
		},
		{
			"Valid Json",
			response{
				Subscription: &subscription{
					ID: "123",
					Endpoint: flare.SubscriptionEndpoint{
						URL:     url.URL{Scheme: "http", Host: "app.io", Path: "/update"},
						Method:  http.MethodPost,
						Headers: map[string][]string{"Content-Type": {"application/json"}},
					},
					Delivery: flare.SubscriptionDelivery{
						Success: []int{200},
						Discard: []int{500},
					},
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				},
			},
			load("responseMarshalJSON.valid3.json"),
			false,
		},
		{
			"Valid Json",
			response{
				Pagination: &pagination{Limit: 10, Offset: 20, Total: 30},
				Subscriptions: []subscription{
					{
						ID: "123",
						Endpoint: flare.SubscriptionEndpoint{
							URL:     url.URL{Scheme: "http", Host: "app.io", Path: "/update"},
							Method:  http.MethodPost,
							Headers: map[string][]string{"Content-Type": {"application/json"}},
						},
						Delivery: flare.SubscriptionDelivery{
							Success: []int{200},
							Discard: []int{500},
						},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
					{
						ID: "456",
						Endpoint: flare.SubscriptionEndpoint{
							URL:     url.URL{Scheme: "https", Host: "app.com", Path: "/update"},
							Method:  http.MethodPost,
							Headers: map[string][]string{"Content-Type": {"application/json"}},
						},
						Delivery: flare.SubscriptionDelivery{
							Success: []int{200},
							Discard: []int{500},
						},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
				},
			},
			load("responseMarshalJSON.valid4.json"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("response.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			c1, c2 := make(map[string]interface{}), make(map[string]interface{})
			if err := json.Unmarshal([]byte(content), &c1); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c1, content,
				)))
				t.FailNow()
			}

			if err := json.Unmarshal(tt.output, &c2); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c2, string(tt.output),
				)))
				t.FailNow()
			}

			if !reflect.DeepEqual(c1, c2) {
				t.Errorf("response.MarshalJSON invalid result, want '%v', got '%v'", c2, c1)
			}
		})
	}
}

func TestSubscriptionCreateValid(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		hasErr bool
	}{
		{
			"Valid",
			load("subscriptionCreateValid.valid1.json"),
			false,
		},
		{
			"Missing URL",
			load("subscriptionCreateValid.invalid1.json"),
			true,
		},
		{
			"Invalid HTTP method",
			load("subscriptionCreateValid.invalid2.json"),
			true,
		},
		{
			"Missing delivery success",
			load("subscriptionCreateValid.invalid3.json"),
			true,
		},
		{
			"Missing selivery discard",
			load("subscriptionCreateValid.invalid4.json"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var content subscriptionCreate
			if err := json.Unmarshal([]byte(tt.input), &content); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", content, string(tt.input),
				)))
				t.FailNow()
			}

			err := content.valid()
			if tt.hasErr != (err != nil) {
				t.Errorf(
					"subscriptionCreate.valid invalid result, want '%v', got '%v'", tt.hasErr, (err != nil),
				)
				t.FailNow()
			}
		})
	}
}

func TestSubscriptionToFlareSubscription(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		output *flare.Subscription
		hasErr bool
	}{
		{
			"Invalid URL",
			load("subscriptionToFlareSubscription.invalid.json"),
			nil,
			true,
		},
		{
			"Valid",
			load("subscriptionToFlareSubscription.valid.json"),
			&flare.Subscription{
				Delivery: flare.SubscriptionDelivery{
					Discard: []int{500},
					Success: []int{200},
				},
				Endpoint: flare.SubscriptionEndpoint{
					URL:    url.URL{Scheme: "http", Host: "app.io", Path: "/update"},
					Method: "post",
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var content subscriptionCreate
			if err := json.Unmarshal([]byte(tt.input), &content); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", content, string(tt.input),
				)))
				t.FailNow()
			}

			result, err := content.toFlareSubscription()
			if tt.hasErr != (err != nil) {
				t.Errorf(
					"subscriptionCreate.toFlareSubscription invalid result, want '%v', got '%v'",
					tt.hasErr, (err != nil),
				)
				t.FailNow()
			}
			if tt.hasErr {
				return
			}

			result.ID = ""
			if !reflect.DeepEqual(result, tt.output) {
				t.Errorf(
					"subscriptionCreate.toFlareSubscription invalid result, want '%v', got '%v'",
					tt.output, result,
				)
			}
		})
	}
}

func load(name string) []byte {
	path := fmt.Sprintf("testdata/%s", name)
	f, err := os.Open(path)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during open '%s'", path)))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during read '%s'", path)))
	}
	return content
}
