// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	httpTest "github.com/diegobernardes/flare/infra/http/test"
	infraTest "github.com/diegobernardes/flare/infra/test"
	"github.com/diegobernardes/flare/repository/memory"
	"github.com/diegobernardes/flare/repository/test"
)

func TestNewService(t *testing.T) {
	Convey("Given a list of valid service options", t, func() {
		writer, err := infraHTTP.NewWriter(log.NewNopLogger())
		So(err, ShouldBeNil)

		tests := [][]func(*Service){
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionURI(func(string, string) string { return "" }),
				ServiceParsePagination(infraHTTP.ParsePagination(30)),
				ServiceWriter(writer),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				_, err := NewService(tt...)
				So(err, ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid service options", t, func() {
		tests := [][]func(*Service){
			{},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
			},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
			},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
			},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionID(func(*http.Request) string { return "" }),
			},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionURI(func(string, string) string { return "" }),
			},
			{
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionID(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionURI(func(string, string) string { return "" }),
				ServiceParsePagination(infraHTTP.ParsePagination(30)),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				_, err := NewService(tt...)
				So(err, ShouldNotBeNil)
			}
		})
	})
}

func TestServiceHandleIndex(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title                  string
			req                    *http.Request
			status                 int
			header                 http.Header
			body                   []byte
			subscriptionRepository flare.SubscriptionRepositorier
			resourceRepository     flare.ResourceRepositorier
		}{
			{
				"The request should have a invalid pagination 1",
				httptest.NewRequest("GET", "http://resources/123/subscriptions?limit=sample", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.invalidPagination.1.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The request should have a invalid pagination 2",
				httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=sample", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.invalidPagination.2.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The request should have a invalid pagination 3",
				httptest.NewRequest("GET", "http://resources/123/subscriptions?limit=-1", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.invalidPagination.3.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The request should have a invalid pagination 4",
				httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=-1", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.invalidPagination.4.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should be a resource repository error",
				httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.resourceRepositoryError.json"),
				test.NewSubscription(),
				test.NewResource(test.ResourceError(errors.New("error during repository search"))),
			},
			{
				"The response should be a subscription repository error",
				httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.subscriptionRepositoryError.json"),
				test.NewSubscription(test.SubscriptionError(
					errors.New("error during repository search"),
				)),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleIndex.inputResource.json")),
					test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
			},
			{
				"The response should be a subscription not found",
				httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.resourceNotFound.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should be a empty list of subscriptions",
				httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.emptySearch.json"),
				test.NewSubscription(
					test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleIndex.inputResource.json")),
					test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
			},
			{
				"The result should contains subscriptions and the pagination",
				httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=1", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleIndex.valid.json"),
				test.NewSubscription(
					test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					test.SubscriptionLoadSliceByteSubscription(
						infraTest.Load("serviceHandleIndex.inputSubscription.json"),
					),
				),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleIndex.inputResource.json")),
					test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceSubscriptionRepository(tt.subscriptionRepository),
					ServiceResourceRepository(tt.resourceRepository),
					ServiceGetResourceID(func(r *http.Request) string { return "123" }),
					ServiceGetSubscriptionID(func(r *http.Request) string { return "" }),
					ServiceGetSubscriptionURI(func(reId, subId string) string { return "" }),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)
				httpTest.Runner(tt.status, tt.header, service.HandleIndex, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleShow(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title                  string
			req                    *http.Request
			status                 int
			header                 http.Header
			body                   []byte
			subscriptionRepository flare.SubscriptionRepositorier
			resourceRepository     flare.ResourceRepositorier
		}{
			{
				"The response should be a subscription not found",
				httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleShow.notFound.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should be a subscription",
				httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleShow.valid.output.json"),
				test.NewSubscription(
					test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					test.SubscriptionLoadSliceByteSubscription(
						infraTest.Load("serviceHandleShow.valid.input.json"),
					),
				),
				test.NewResource(),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceSubscriptionRepository(tt.subscriptionRepository),
					ServiceResourceRepository(tt.resourceRepository),
					ServiceGetResourceID(func(r *http.Request) string { return "123" }),
					ServiceGetSubscriptionID(func(r *http.Request) string { return "456" }),
					ServiceGetSubscriptionURI(func(reId, subId string) string {
						return fmt.Sprintf("http://resources/%s/subscriptions/%s", reId, subId)
					}),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)
				httpTest.Runner(tt.status, tt.header, service.HandleShow, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleDelete(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title                  string
			req                    *http.Request
			status                 int
			header                 http.Header
			body                   []byte
			subscriptionRepository flare.SubscriptionRepositorier
			resourceRepository     flare.ResourceRepositorier
		}{
			{
				"The response should be a subscription not found",
				httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleDelete.notFound.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should be the result of a deleted subscription",
				httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
				http.StatusNoContent,
				http.Header{},
				nil,
				test.NewSubscription(
					test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					test.SubscriptionLoadSliceByteSubscription(
						infraTest.Load("serviceHandleShow.valid.input.json"),
					),
				),
				test.NewResource(),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceSubscriptionRepository(tt.subscriptionRepository),
					ServiceResourceRepository(tt.resourceRepository),
					ServiceGetResourceID(func(r *http.Request) string { return "123" }),
					ServiceGetSubscriptionID(func(r *http.Request) string { return "456" }),
					ServiceGetSubscriptionURI(func(reId, subId string) string { return "" }),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)
				httpTest.Runner(tt.status, tt.header, service.HandleDelete, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleCreate(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title                  string
			req                    *http.Request
			status                 int
			header                 http.Header
			body                   []byte
			subscriptionRepository flare.SubscriptionRepositorier
			resourceRepository     flare.ResourceRepositorier
		}{
			{
				"The response should have a invalid resource 1",
				httptest.NewRequest(http.MethodPost, "http://resources/123/subscriptions", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleCreate.invalid.1.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should have a invalid resource 2",
				httptest.NewRequest(
					http.MethodPost, "http://resources/123/subscriptions", bytes.NewBufferString("{}"),
				),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleCreate.invalid.2.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should have a invalid resource 3",
				httptest.NewRequest(
					http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
						infraTest.Load("serviceHandleCreate.invalid.input.json"),
					),
				),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleCreate.invalid.3.json"),
				test.NewSubscription(),
				test.NewResource(),
			},
			{
				"The response should be a subscription repository error",
				httptest.NewRequest(
					http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
						infraTest.Load("serviceHandleCreate.input.json"),
					),
				),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleCreate.subscriptionRepositoryError.json"),
				test.NewSubscription(
					test.SubscriptionError(errors.New("error at repository")),
				),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleCreate.resourceInput.json")),
				),
			},
			{
				"The response should be a subscription conflict",
				httptest.NewRequest(
					http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
						infraTest.Load("serviceHandleCreate.input.json"),
					),
				),
				http.StatusConflict,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleCreate.subscriptionRepositoryConflict.json"),
				test.NewSubscription(
					test.SubscriptionCreateId("456"),
					test.SubscriptionLoadSliceByteSubscription(
						infraTest.Load("serviceHandleCreate.inputArray.json"),
					),
				),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleCreate.resourceInput.json")),
				),
			},
			{
				"The response should be the result of a created subscription",
				httptest.NewRequest(
					http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
						infraTest.Load("serviceHandleCreate.input.json"),
					),
				),
				http.StatusCreated,
				http.Header{
					"Content-Type": []string{"application/json"},
					"Location":     []string{"http://resources/123/subscriptions/456"},
				},
				infraTest.Load("serviceHandleCreate.create.json"),
				test.NewSubscription(
					test.SubscriptionCreateId("456"),
					test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
				test.NewResource(
					test.ResourceLoadSliceByteResource(infraTest.Load("serviceHandleCreate.resourceInput.json")),
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceSubscriptionRepository(tt.subscriptionRepository),
					ServiceResourceRepository(tt.resourceRepository),
					ServiceGetResourceID(func(r *http.Request) string { return "123" }),
					ServiceGetSubscriptionID(func(r *http.Request) string { return "456" }),
					ServiceGetSubscriptionURI(func(reId, subId string) string {
						return fmt.Sprintf("http://resources/%s/subscriptions/%s", reId, subId)
					}),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)
				httpTest.Runner(tt.status, tt.header, service.HandleCreate, tt.req, tt.body)
			})
		}
	})
}
