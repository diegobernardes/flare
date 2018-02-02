// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

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
	Convey("Feature: Marshal to JSON a Pagination", t, func() {
		Convey("Given a list of valid paginations", func() {
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

			Convey("Should return a valid JSON", func() {
				for _, tt := range tests {
					content, err := tt.input.MarshalJSON()
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, tt.output)
				}
			})
		})
	})
}

func TestResponseMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal to JSON a response", t, func() {
		Convey("Given a list of valid responses", func() {
			tests := []struct {
				input  response
				output []byte
			}{
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
					infraTest.Load("response.marshalJSON.valid.1.json"),
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
					infraTest.Load("response.marshalJSON.valid.2.json"),
				},
			}

			Convey("Should return a valid JSON", func() {
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
	})
}

func TestSubscriptionCreateValid(t *testing.T) {
	Convey("Feature: Should validate the subscriptionCreate", t, func() {
		Convey("Given a list of valid subscriptionCreate", func() {
			tests := [][]byte{
				infraTest.Load("subscriptionCreate.valid.valid.1.json"),
				infraTest.Load("subscriptionCreate.valid.valid.2.json"),
			}

			Convey("Should be valid", func() {
				for _, tt := range tests {
					var content subscriptionCreate
					err := json.Unmarshal([]byte(tt), &content)
					So(err, ShouldBeNil)

					err = content.valid()
					So(err, ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid subscriptionCreate", func() {
			tests := []struct {
				title string
				input []byte
			}{
				{
					"Should be missing the URL",
					infraTest.Load("subscriptionCreate.valid.missingURL.json"),
				},
				{
					"Should have a invalid HTTP method",
					infraTest.Load("subscriptionCreate.valid.invalidHTTPMethod.json"),
				},
				{
					"Should be missing delivery success",
					infraTest.Load("subscriptionCreate.valid.missingDeliverySuccess.json"),
				},
				{
					"Should be missing delivery discard",
					infraTest.Load("subscriptionCreate.valid.missingDeliveryDiscard.json"),
				},
				{
					"Should have a invalid envelope",
					infraTest.Load("subscriptionCreate.valid.invalidEnvelope.json"),
				},
				{
					"Should not have data if skipEnvelope is true",
					infraTest.Load("subscriptionCreate.valid.noDataIfSkipEnvelope.json"),
				},
				{
					"Should have a invalid data",
					infraTest.Load("subscriptionCreate.valid.invalidData.json"),
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
	})
}

func TestSubscriptionCreateToFlareSubscription(t *testing.T) {
	Convey("Feature: Transform subscriptionCreate to flare.Subscription", t, func() {
		Convey("Given a list of valid subscriptionCreate", func() {
			tests := []struct {
				input  []byte
				output *flare.Subscription
			}{
				{
					infraTest.Load("subscriptionCreate.toFlareSubscription.valid.1.json"),
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
				{
					infraTest.Load("subscriptionCreate.toFlareSubscription.valid.2.json"),
					&flare.Subscription{
						Delivery: flare.SubscriptionDelivery{
							Discard: []int{500},
							Success: []int{200},
						},
						Endpoint: flare.SubscriptionEndpoint{
							URL:    url.URL{Scheme: "http", Host: "app.io", Path: "/update"},
							Method: "post",
						},
						SendDocument: true,
					},
				},
			}

			Convey("Should have a valid list of flare.Subscription", func() {
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

		Convey("Given a list of invalid subscriptionCreate", func() {
			tests := []struct {
				input  []byte
				output *flare.Subscription
			}{
				{
					infraTest.Load("subscriptionCreate.toFlareSubscription.invalid.json"),
					nil,
				},
			}

			Convey("Should return a error", func() {
				for _, tt := range tests {
					var content subscriptionCreate
					err := json.Unmarshal([]byte(tt.input), &content)
					So(err, ShouldBeNil)

					_, err = content.toFlareSubscription()
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestSubscriptionCreateValidData(t *testing.T) {
	Convey("Feature: Check if the subscriptionCreate has a valid data", t, func() {
		Convey("Given a list of valid subscriptionCreate", func() {
			tests := []subscriptionCreate{
				{},
				{Data: map[string]interface{}{"service": "user"}},
				{Data: map[string]interface{}{"rate": float64(1)}},
				{Data: map[string]interface{}{"enable": true}},
				{Data: map[string]interface{}{"service": "user", "rate": float64(1), "enable": true}},
				{Data: map[string]interface{}{"service": "user", "rate": float64(1), "enable": true}},
				{
					Data: map[string]interface{}{
						"service": "user",
						"object": []interface{}{
							"sample", float64(2),
						},
					},
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(tt.validData(), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid subscriptionCreate", func() {
			tests := []subscriptionCreate{
				{Data: map[string]interface{}{"object": map[string]interface{}{}}},
				{Data: map[string]interface{}{"object": []interface{}{map[string]interface{}{}}}},
			}

			Convey("Should return a error", func() {
				for _, tt := range tests {
					So(tt.validData(), ShouldNotBeNil)
				}
			})
		})
	})
}
