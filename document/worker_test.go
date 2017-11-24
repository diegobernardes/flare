// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
	repoTest "github.com/diegobernardes/flare/repository/test"
	subscriptionTest "github.com/diegobernardes/flare/subscription/test"
)

func TestWorkerMarshal(t *testing.T) {
	Convey("Given a list of valid params", t, func() {
		tests := []struct {
			action   string
			document flare.Document
			output   []byte
		}{
			{
				flare.SubscriptionTriggerDelete,
				flare.Document{
					ID:        "123",
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource:  flare.Resource{ID: "456"},
				},
				infraTest.Load("workerMarshal.output.1.json"),
			},
			{
				flare.SubscriptionTriggerUpdate,
				flare.Document{
					ID:        "123",
					Revision:  10,
					Content:   map[string]interface{}{"resource": "user"},
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource: flare.Resource{
						ID: "456",
						Change: flare.ResourceChange{
							Format: "2006-01-02",
						},
					},
				},
				infraTest.Load("workerMarshal.output.2.json"),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				content, err := w.marshal(tt.action, &tt.document)
				So(err, ShouldBeNil)

				b1, b2 := make(map[string]interface{}), make(map[string]interface{})
				err = json.Unmarshal(content, &b1)
				So(err, ShouldBeNil)

				err = json.Unmarshal(tt.output, &b2)
				So(err, ShouldBeNil)

				So(b2, ShouldResemble, b1)
			}
		})
	})
}

func TestWorkerUnmarshal(t *testing.T) {
	Convey("Given a list of valid params", t, func() {
		tests := []struct {
			input  []byte
			action string
			output flare.Document
		}{
			{
				infraTest.Load("workerMarshal.output.1.json"),
				flare.SubscriptionTriggerDelete,
				flare.Document{
					ID:        "123",
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource:  flare.Resource{ID: "456"},
				},
			},
			{
				infraTest.Load("workerMarshal.output.2.json"),
				flare.SubscriptionTriggerUpdate,
				flare.Document{
					ID:        "123",
					Revision:  10,
					Content:   map[string]interface{}{"resource": "user"},
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource: flare.Resource{
						ID: "456",
						Change: flare.ResourceChange{
							Format: "2006-01-02",
						},
					},
				},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				action, document, err := w.unmarshal(tt.input)
				So(err, ShouldBeNil)
				So(action, ShouldEqual, tt.action)
				So(*document, ShouldResemble, tt.output)
			}
		})
	})
}

func TestWorkerPush(t *testing.T) {
	Convey("Given a list of valid documents", t, func() {
		tests := []struct {
			action   string
			document flare.Document
		}{
			{
				flare.SubscriptionTriggerDelete,
				flare.Document{
					ID:        "123",
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource:  flare.Resource{ID: "456"},
				},
			},
			{
				flare.SubscriptionTriggerUpdate,
				flare.Document{
					ID:        "123",
					Revision:  10,
					Content:   map[string]interface{}{"resource": "user"},
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource: flare.Resource{
						ID: "456",
						Change: flare.ResourceChange{
							Format: "2006-01-02",
						},
					},
				},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				w.Init(WorkerPusher(newPushWorkerMock(nil)))
				So(w.push(context.Background(), tt.action, &tt.document), ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid documents", t, func() {
		tests := []struct {
			action   string
			document flare.Document
		}{
			{
				flare.SubscriptionTriggerDelete,
				flare.Document{
					ID:        "123",
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Resource:  flare.Resource{ID: "456"},
				},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				w.Init(WorkerPusher(newPushWorkerMock(errors.New("error during push"))))
				So(w.push(context.Background(), tt.action, &tt.document), ShouldNotBeNil)
			}
		})
	})
}

func TestWorkerProcess(t *testing.T) {
	Convey("Given a list of valid params", t, func() {
		tests := [][]byte{
			infraTest.Load("workerMarshal.output.1.json"),
			infraTest.Load("workerMarshal.output.2.json"),
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				w.Init(
					WorkerDocumentRepository(repoTest.NewDocument()),
					WorkerPusher(newPushWorkerMock(nil)),
					WorkerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
				)
				So(w.Process(context.Background(), tt), ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid params", t, func() {
		tests := []struct {
			input    []byte
			document flare.DocumentRepositorier
			trigger  flare.SubscriptionTrigger
		}{
			{
				infraTest.Load("workerMarshal.output.1.json"),
				repoTest.NewDocument(),
				subscriptionTest.NewTrigger(errors.New("error during push")),
			},
			{
				infraTest.Load("workerMarshal.output.2.json"),
				repoTest.NewDocument(repoTest.DocumentError(errors.New("error at repository"))),
				subscriptionTest.NewTrigger(nil),
			},
			{
				infraTest.Load("workerMarshal.output.2.json"),
				repoTest.NewDocument(),
				subscriptionTest.NewTrigger(errors.New("error during push")),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				w.Init(
					WorkerDocumentRepository(tt.document),
					WorkerPusher(newPushWorkerMock(nil)),
					WorkerSubscriptionTrigger(tt.trigger),
				)
				So(w.Process(context.Background(), tt.input), ShouldNotBeNil)
			}
		})
	})
}

func TestWorkerInit(t *testing.T) {
	Convey("Given a list of valid worker options", t, func() {
		tests := [][]func(*Worker){
			{
				WorkerDocumentRepository(repoTest.NewDocument()),
				WorkerPusher(newPushWorkerMock(nil)),
				WorkerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
			},
		}

		Convey("The service initialization should not return error", func() {
			for _, tt := range tests {
				w := &Worker{}
				So(w.Init(tt...), ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid worker options", t, func() {
		tests := [][]func(*Worker){
			{},
			{
				WorkerPusher(newPushWorkerMock(nil)),
			},
			{
				WorkerPusher(newPushWorkerMock(nil)),
				WorkerDocumentRepository(repoTest.NewDocument()),
			},
		}

		Convey("The service initialization should return error", func() {
			for _, tt := range tests {
				w := &Worker{}
				So(w.Init(tt...), ShouldNotBeNil)
			}
		})
	})
}
