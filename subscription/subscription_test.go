// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
)

func TestPaginationMarshalJSON(t *testing.T) {
	Convey("Given a list of valid paginations", t, func() {
		tests := []struct {
			input  pagination
			output string
		}{
			{
				pagination{Limit: 30, Offset: 0},
				`{"limit":30,"offset":0,"total":0}`,
			},
			{
				pagination{Limit: 10, Offset: 30, Total: 120},
				`{"limit":10,"offset":30,"total":120}`,
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(content), ShouldEqual, tt.output)
			}
		})
	})
}

func TestResponseMarshalJSON(t *testing.T) {
	Convey("Given a list of valid responses", t, func() {
		tests := []struct {
			input  response
			output []byte
		}{
			{
				response{
					Error: &responseError{
						Title:  "error during query",
						Detail: "detail from error",
					},
				},
				infraTest.Load("responseMarshalJSON.invalid.1.json"),
			},
			{
				response{
					Error: &responseError{
						Title: "service unavailable",
					},
				},
				infraTest.Load("responseMarshalJSON.invalid.2.json"),
			},
			{
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
				infraTest.Load("responseMarshalJSON.valid.1.json"),
			},
			{
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
				infraTest.Load("responseMarshalJSON.valid.2.json"),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)

				c1, c2 := make(map[string]interface{}), make(map[string]interface{})
				err = json.Unmarshal([]byte(content), &c1)
				So(err, ShouldBeNil)

				err = json.Unmarshal([]byte(tt.output), &c2)
				So(err, ShouldBeNil)

				So(c1, ShouldResemble, c2)
			}
		})
	})
}

func TestSubscriptionCreateValid(t *testing.T) {
	Convey("Given a list of valid subscriptionCreate", t, func() {
		tests := [][]byte{
			infraTest.Load("subscriptionCreateValid.valid.json"),
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				var content subscriptionCreate
				err := json.Unmarshal([]byte(tt), &content)
				So(err, ShouldBeNil)

				err = content.valid()
				So(err, ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid subscriptionCreate", t, func() {
		tests := []struct {
			title string
			input []byte
		}{
			{
				"Should be missing the URL",
				infraTest.Load("subscriptionCreateValid.invalid.1.json"),
			},
			{
				"Should have a invalid HTTP method",
				infraTest.Load("subscriptionCreateValid.invalid.2.json"),
			},
			{
				"Should be missing delivery success",
				infraTest.Load("subscriptionCreateValid.invalid.3.json"),
			},
			{
				"Should be missing delivery discard",
				infraTest.Load("subscriptionCreateValid.invalid.4.json"),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				var content subscriptionCreate
				err := json.Unmarshal([]byte(tt.input), &content)
				So(err, ShouldBeNil)

				err = content.valid()
				So(err, ShouldNotBeNil)
			})
		}
	})
}

func TestSubscriptionToFlareSubscription(t *testing.T) {
	Convey("Given a list of valid flare.Subscription", t, func() {
		tests := []struct {
			input  []byte
			output *flare.Subscription
		}{
			{
				infraTest.Load("subscriptionToFlareSubscription.valid.json"),
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
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				var content subscriptionCreate
				err := json.Unmarshal([]byte(tt.input), &content)
				So(err, ShouldBeNil)

				result, err := content.toFlareSubscription()
				So(err, ShouldBeNil)
				result.ID = ""

				So(result, ShouldResemble, tt.output)
			}
		})
	})

	Convey("Given a list of invalid flare.Subscription", t, func() {
		tests := []struct {
			input  []byte
			output *flare.Subscription
		}{
			{
				infraTest.Load("subscriptionToFlareSubscription.invalid.json"),
				nil,
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				var content subscriptionCreate
				err := json.Unmarshal([]byte(tt.input), &content)
				So(err, ShouldBeNil)

				_, err = content.toFlareSubscription()
				So(err, ShouldNotBeNil)
			}
		})
	})
}
