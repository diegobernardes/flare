// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
	testQueue "github.com/diegobernardes/flare/provider/test/queue"
	testRepository "github.com/diegobernardes/flare/provider/test/repository"
)

func TestInitPartition(t *testing.T) {
	Convey("Feature: Initialize a instance of Partition", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := [][]func(*Partition){
				{
					PartitionPusher(testQueue.NewClient()),
					PartitionResourceRepository(&testRepository.Resource{}),
					PartitionOutput(&partitionOutput{}),
				},
				{
					PartitionPusher(testQueue.NewClient()),
					PartitionResourceRepository(&testRepository.Resource{}),
					PartitionOutput(&partitionOutput{}),
					PartitionConcurrency(5),
				},
			}

			Convey("Should not output a error", func() {
				for _, tt := range tests {
					var p Partition
					So(p.Init(tt...), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]func(*Partition){
				{},
				{
					PartitionConcurrency(-1),
				},
				{
					PartitionResourceRepository(&testRepository.Resource{}),
				},
				{
					PartitionResourceRepository(&testRepository.Resource{}),
					PartitionPusher(testQueue.NewClient()),
				},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					var p Partition
					So(p.Init(tt...), ShouldNotBeNil)
				}
			})
		})
	})
}

func TestPartitionMarshal(t *testing.T) {
	Convey("Feature: Marshal message to JSON", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document *flare.Document
				action   string
				expected []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					infraTest.Load("partition.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					infraTest.Load("partition.marshal.2.json"),
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					var p Partition
					content, err := p.marshal(tt.document, tt.action)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(content, tt.expected)
				}
			})
		})
	})
}

func TestPartitionUnmarshal(t *testing.T) {
	Convey("Feature: Unmarshal JSON to message", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				expected *flare.Document
				action   string
				input    []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					infraTest.Load("partition.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					infraTest.Load("partition.marshal.2.json"),
				},
			}

			Convey("Should output a valid message", func() {
				for _, tt := range tests {
					var p Partition
					document, action, err := p.unmarshal(tt.input)
					So(err, ShouldBeNil)
					So(action, ShouldEqual, tt.action)
					So(*document, ShouldResemble, *tt.expected)
				}
			})

			Convey("Given a list of invalid parameters", func() {
				tests := [][]byte{
					{},
					[]byte("{"),
				}

				Convey("Should output a error", func() {
					for _, tt := range tests {
						var p Partition
						_, _, err := p.unmarshal(tt)
						So(err, ShouldNotBeNil)
					}
				})
			})
		})
	})
}

func TestPartitionPush(t *testing.T) {
	Convey("Feature: Push the message to be processed", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document *flare.Document
				action   string
				expected []byte
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					infraTest.Load("partition.marshal.1.json"),
				},
				{
					&flare.Document{ID: "3", Resource: flare.Resource{ID: "4"}},
					flare.SubscriptionTriggerDelete,
					infraTest.Load("partition.marshal.2.json"),
				},
			}

			Convey("Should push the message", func() {
				for _, tt := range tests {
					q := testQueue.NewClient()
					p := Partition{pusher: q}
					err := p.Push(context.Background(), tt.document, tt.action)
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(q.Content, tt.expected)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := []struct {
				document *flare.Document
				action   string
				err      error
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					errors.New("error during push"),
				},
			}

			Convey("Should have a error during message push", func() {
				for _, tt := range tests {
					q := testQueue.NewClient(testQueue.ClientError(tt.err))
					p := Partition{pusher: q}
					err := p.Push(context.Background(), tt.document, tt.action)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestPartitionProcess(t *testing.T) {
	Convey("Feature: Process the message", t, func() {
		Convey("Given a list of valid parameters", func() {
			tests := []struct {
				document          *flare.Document
				action            string
				repositoryOptions []func(*testRepository.Resource)
				messages          []partitionOutputMessage
			}{
				{
					&flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
					flare.SubscriptionTriggerUpdate,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.1.json")),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.ResourcePartitions([]string{"1", "2", "3"}),
					},
					[]partitionOutputMessage{
						{
							document:  flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
							action:    flare.SubscriptionTriggerUpdate,
							partition: "1",
						},
						{
							document:  flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
							action:    flare.SubscriptionTriggerUpdate,
							partition: "2",
						},
						{
							document:  flare.Document{ID: "1", Resource: flare.Resource{ID: "2"}},
							action:    flare.SubscriptionTriggerUpdate,
							partition: "3",
						},
					},
				},
			}

			Convey("Should process the message", func() {
				for _, tt := range tests {
					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)

					o := &partitionOutput{}
					p := Partition{
						repository:         repo.Resource(),
						concurrencyControl: make(chan struct{}, 1),
						output:             o,
					}
					content, err := p.marshal(tt.document, tt.action)
					So(err, ShouldBeNil)
					So(p.Process(context.Background(), content), ShouldBeNil)
					So(tt.messages, ShouldResemble, o.messages)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := []struct {
				document          []byte
				action            string
				repositoryOptions []func(*testRepository.Resource)
				outputError       error
			}{
				{
					[]byte(`{`),
					flare.SubscriptionTriggerUpdate,
					nil,
					nil,
				},
				{
					[]byte(`{}`),
					flare.SubscriptionTriggerUpdate,
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error during search")),
					},
					nil,
				},
				{
					[]byte(`{"resourceID": "2"}`),
					flare.SubscriptionTriggerUpdate,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.1.json")),
						testRepository.ResourcePartitions([]string{"1"}),
					},
					errors.New("error on output push"),
				},
			}

			Convey("Should have a error", func() {
				for _, tt := range tests {
					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)

					o := &partitionOutput{err: tt.outputError}
					p := Partition{
						repository:         repo.Resource(),
						concurrencyControl: make(chan struct{}, 1),
						output:             o,
					}

					So(p.Process(context.Background(), tt.document), ShouldNotBeNil)
				}
			})
		})
	})
}

type partitionOutput struct {
	mutex    sync.Mutex
	messages []partitionOutputMessage
	err      error
}

func (p *partitionOutput) Push(
	ctx context.Context, d *flare.Document, action, partition string,
) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.err != nil {
		return p.err
	}

	p.messages = append(p.messages, partitionOutputMessage{
		document:  *d,
		action:    action,
		partition: partition,
	})
	return nil
}

type partitionOutputMessage struct {
	document  flare.Document
	action    string
	partition string
}
