// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
	memoryRepository "github.com/diegobernardes/flare/provider/memory/repository"
	testQueue "github.com/diegobernardes/flare/provider/test/queue"
	testRepository "github.com/diegobernardes/flare/provider/test/repository"
)

func TestInitSpread(t *testing.T) {
	Convey("Feature: Initialize a instance of Spread", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := [][]func(*Spread){
				{
					SpreadPusher(testQueue.NewClient()),
					SpreadSubscriptionRepository(&testRepository.Subscription{}),
					SpreadOutput(&spreadOutput{}),
				},
				{
					SpreadPusher(testQueue.NewClient()),
					SpreadSubscriptionRepository(&testRepository.Subscription{}),
					SpreadOutput(&spreadOutput{}),
					SpreadConcurrency(5),
				},
			}

			Convey("Should not output a error", func() {
				for _, tt := range tests {
					var s Spread
					So(s.Init(tt...), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]func(*Spread){
				{},
				{
					SpreadConcurrency(-1),
				},
				{
					SpreadPusher(testQueue.NewClient()),
				},
				{
					SpreadPusher(testQueue.NewClient()),
					SpreadSubscriptionRepository(&testRepository.Subscription{}),
				},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					var s Spread
					So(s.Init(tt...), ShouldNotBeNil)
				}
			})
		})
	})
}

func TestSpreadMarshal(t *testing.T) {
	Convey("Feature: Marshal message to JSON", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document  *flare.Document
				action    string
				partition string
				expected  []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					"1",
					infraTest.Load("spread.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					"2",
					infraTest.Load("spread.marshal.2.json"),
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					var s Spread
					content, err := s.marshal(tt.document, tt.action, tt.partition)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(content, tt.expected)
				}
			})
		})
	})
}

func TestSpreadUnmarshal(t *testing.T) {
	Convey("Feature: Unmarshal JSON to message", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document  *flare.Document
				action    string
				partition string
				input     []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					"1",
					infraTest.Load("spread.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					"2",
					infraTest.Load("spread.marshal.2.json"),
				},
			}

			Convey("Should output a valid message", func() {
				for _, tt := range tests {
					var s Spread
					document, action, partition, err := s.unmarshal(tt.input)
					So(err, ShouldBeNil)
					So(action, ShouldEqual, tt.action)
					So(partition, ShouldEqual, tt.partition)
					So(*document, ShouldResemble, *tt.document)
				}
			})
		})

		Convey("Given a list of invalid serialized documents and actions", func() {
			tests := [][]byte{
				infraTest.Load("spread.unmarshal.invalid.1.json"),
				infraTest.Load("spread.unmarshal.invalid.2.json"),
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					var s Spread
					_, _, _, err := s.unmarshal(tt)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestSpreadPush(t *testing.T) {
	Convey("Feature: Push the message to be processed", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document  *flare.Document
				action    string
				partition string
				expected  []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					"1",
					infraTest.Load("spread.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					"2",
					infraTest.Load("spread.marshal.2.json"),
				},
			}

			Convey("Should push the message", func() {
				for _, tt := range tests {
					q := testQueue.NewClient()
					s := Spread{pusher: q}
					err := s.Push(context.Background(), tt.document, tt.action, tt.partition)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(q.Content, tt.expected)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := []struct {
				document  *flare.Document
				action    string
				partition string
				err       error
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					"1",
					errors.New("error during push"),
				},
			}

			Convey("Should have a error during message push", func() {
				for _, tt := range tests {
					q := testQueue.NewClient(testQueue.ClientError(tt.err))
					s := Spread{pusher: q}
					err := s.Push(context.Background(), tt.document, tt.action, tt.partition)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestSpreadProcess(t *testing.T) {
	Convey("Feature: Process the message", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document                      *flare.Document
				action                        string
				subscriptionRepositoryOptions []func(*testRepository.Subscription)
				resourceRepositoryOptions     []func(*testRepository.Resource)
				messages                      []spreadOutputMessage
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "123"}},
					flare.SubscriptionTriggerUpdate,
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionLoadSliceByteSubscription(
							infraTest.Load("spread.subscription.input.subscription.json"),
						),
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("spread.subscription.input.resource.json"),
						),
					},
					[]spreadOutputMessage{
						{
							document: flare.Document{ID: "1", Resource: flare.Resource{ID: "123"}},
							subscription: flare.Subscription{
								ID: "1",
								Endpoint: flare.SubscriptionEndpoint{
									URL:    url.URL{Scheme: "http", Host: "app1.com"},
									Method: http.MethodPost,
								},
								Delivery: flare.SubscriptionDelivery{
									Success: []int{200},
									Discard: []int{500},
								},
								Resource: flare.Resource{ID: "123"},
							},
							action: flare.SubscriptionTriggerUpdate,
						},
					},
				},
			}

			Convey("Should process the message", func() {
				for _, tt := range tests {
					repoBase := memoryRepository.NewClient(
						memoryRepository.ClientResourceOptions(
							memoryRepository.ResourcePartitionLimit(100),
						),
					)
					repo := testRepository.NewClientWithBase(
						repoBase,
						testRepository.ClientResourceOptions(tt.resourceRepositoryOptions...),
						testRepository.ClientSubscriptionOptions(tt.subscriptionRepositoryOptions...),
					)

					partitions, err := repo.Resource().Partitions(context.Background(), "123")
					So(err, ShouldBeNil)
					So(partitions, ShouldHaveLength, 1)

					for i := range tt.messages {
						tt.messages[i].subscription.Partition = partitions[0]
					}

					o := &spreadOutput{}
					s := Spread{
						repository:         repo.Subscription(),
						concurrencyControl: make(chan struct{}, 1),
						output:             o,
					}
					content, err := s.marshal(tt.document, tt.action, partitions[0])
					So(err, ShouldBeNil)
					So(s.Process(context.Background(), content), ShouldBeNil)

					for i := range o.messages {
						o.messages[i].subscription.CreatedAt = time.Time{}
					}
					So(o.messages, ShouldResemble, tt.messages)
				}
			})
		})
	})
}

type spreadOutput struct {
	err      error
	mutex    sync.Mutex
	messages []spreadOutputMessage
}

func (s *spreadOutput) Push(
	ctx context.Context, subscription *flare.Subscription, document *flare.Document, action string,
) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.err != nil {
		return s.err
	}

	s.messages = append(s.messages, spreadOutputMessage{
		document:     *document,
		subscription: *subscription,
		action:       action,
	})
	return nil
}

type spreadOutputMessage struct {
	document     flare.Document
	subscription flare.Subscription
	action       string
}
