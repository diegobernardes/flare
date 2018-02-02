// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
	testQueue "github.com/diegobernardes/flare/provider/test/queue"
	testRepository "github.com/diegobernardes/flare/provider/test/repository"
)

func TestInitDelivery(t *testing.T) {
	Convey("Feature: Initialize a instance of Delivery", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := [][]func(*Delivery){
				{
					DeliveryPusher(testQueue.NewClient()),
					DeliverySubscriptionRepository(&testRepository.Subscription{}),
					DeliveryHTTPClient(http.DefaultClient),
				},
			}

			Convey("Should not output a error", func() {
				for _, tt := range tests {
					var d Delivery
					So(d.Init(tt...), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]func(*Delivery){
				{},
				{
					DeliveryPusher(testQueue.NewClient()),
				},
				{
					DeliveryPusher(testQueue.NewClient()),
					DeliverySubscriptionRepository(&testRepository.Subscription{}),
				},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					var d Delivery
					So(d.Init(tt...), ShouldNotBeNil)
				}
			})
		})
	})
}

func TestDeliveryMarshal(t *testing.T) {
	Convey("Feature: Marshal message to JSON", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				subscription *flare.Subscription
				document     *flare.Document
				action       string
				expected     string
			}{
				{
					&flare.Subscription{ID: "3"},
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					`{"action": "update", "documentID": "1", "resourceID": "2", "subscriptionID": "3"}`,
				},
				{
					&flare.Subscription{ID: "6"},
					&flare.Document{ID: "4", Resource: flare.Resource{ID: "5"}},
					flare.SubscriptionTriggerDelete,
					`{"action": "delete", "documentID": "4", "resourceID": "5", "subscriptionID": "6"}`,
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					var d Delivery
					content, err := d.marshal(tt.subscription, tt.document, tt.action)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(content, []byte(tt.expected))
				}
			})
		})
	})
}

func TestDeliveryUnmarshal(t *testing.T) {
	Convey("Feature: Unmarshal JSON to message", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				subscription *flare.Subscription
				document     *flare.Document
				action       string
				content      string
			}{
				{
					&flare.Subscription{ID: "3"},
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					`{"action": "update", "documentID": "1", "resourceID": "2", "subscriptionID": "3"}`,
				},
				{
					&flare.Subscription{ID: "6"},
					&flare.Document{ID: "4", Resource: flare.Resource{ID: "5"}},
					flare.SubscriptionTriggerDelete,
					`{"action": "delete", "documentID": "4", "resourceID": "5", "subscriptionID": "6"}`,
				},
			}

			Convey("Should output a valid message", func() {
				for _, tt := range tests {
					var d Delivery
					subscription, document, action, err := d.unmarshal([]byte(tt.content))
					So(err, ShouldBeNil)
					So(action, ShouldEqual, tt.action)
					So(*document, ShouldResemble, *tt.document)
					So(*subscription, ShouldResemble, *tt.subscription)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]byte{
				{},
				[]byte("{"),
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					var d Delivery
					_, _, _, err := d.unmarshal(tt)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestDeliveryPush(t *testing.T) {
	Convey("Feature: Push the message to be processed", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				subscription *flare.Subscription
				document     *flare.Document
				action       string
				expected     string
			}{
				{
					&flare.Subscription{ID: "3"},
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					`{"action": "update", "documentID": "1", "resourceID": "2", "subscriptionID": "3"}`,
				},
				{
					&flare.Subscription{ID: "6"},
					&flare.Document{ID: "4", Resource: flare.Resource{ID: "5"}},
					flare.SubscriptionTriggerDelete,
					`{"action": "delete", "documentID": "4", "resourceID": "5", "subscriptionID": "6"}`,
				},
			}

			Convey("Should push the message", func() {
				for _, tt := range tests {
					q := testQueue.NewClient()
					d := Delivery{pusher: q}
					err := d.Push(context.Background(), tt.subscription, tt.document, tt.action)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(q.Content, []byte(tt.expected))
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := []struct {
				subscription *flare.Subscription
				document     *flare.Document
				action       string
				err          error
			}{
				{
					&flare.Subscription{ID: "3"},
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					errors.New("error during push"),
				},
			}

			Convey("Should have a error during message push", func() {
				for _, tt := range tests {
					q := testQueue.NewClient(testQueue.ClientError(tt.err))
					d := Delivery{pusher: q}
					err := d.Push(context.Background(), tt.subscription, tt.document, tt.action)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}
