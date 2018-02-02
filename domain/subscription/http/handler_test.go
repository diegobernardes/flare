// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

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

	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/http/test"
	infraTest "github.com/diegobernardes/flare/infra/test"
	memory "github.com/diegobernardes/flare/provider/memory/repository"
	testRepository "github.com/diegobernardes/flare/provider/test/repository"
)

func TestNewHandler(t *testing.T) {
	Convey("Feature: Create a new instance of Handler", t, func() {
		Convey("Given a list of valid parameters", func() {
			writer, err := infraHTTP.NewWriter(log.NewNopLogger())
			So(err, ShouldBeNil)

			tests := [][]func(*Handler){
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionURI(func(string, string) string { return "" }),
					HandlerParsePagination(infraHTTP.ParsePagination(30)),
					HandlerWriter(writer),
				},
			}

			Convey("Should output a valid Handler", func() {
				for _, tt := range tests {
					_, err := NewHandler(tt...)
					So(err, ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]func(*Handler){
				{},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
				},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
				},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
				},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionID(func(*http.Request) string { return "" }),
				},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionURI(func(string, string) string { return "" }),
				},
				{
					HandlerSubscriptionRepository(&memory.Subscription{}),
					HandlerResourceRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionID(func(*http.Request) string { return "" }),
					HandlerGetSubscriptionURI(func(string, string) string { return "" }),
					HandlerParsePagination(infraHTTP.ParsePagination(30)),
				},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					_, err := NewHandler(tt...)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestHandlerIndex(t *testing.T) {
	Convey("Feature: Serve a HTTP request to display a list of subscriptions", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title               string
				req                 *http.Request
				status              int
				header              http.Header
				body                []byte
				subscriptionOptions []func(*testRepository.Subscription)
				resourceOptions     []func(*testRepository.Resource)
			}{
				{
					"return a pagination error because of a invalid limit (1)",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions?limit=sample", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.1.json"),
					nil,
					nil,
				},
				{
					"return a pagination error because of a invalid limit (2)",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions?limit=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.3.json"),
					nil,
					nil,
				},
				{
					"return a pagination error because of a invalid offset (1)",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions?offset=sample", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.2.json"),
					nil,
					nil,
				},
				{
					"return a pagination error because of a invalid offset (2)",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions?offset=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.4.json"),
					nil,
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.subscriptionRepositoryError.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionError(errors.New("error during repository search")),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.index.inputResource.json"),
						),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
				{
					"return a empty list of subscriptions",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.emptySearch.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.index.inputResource.json"),
						),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
				{
					"return a list of subscriptions",
					httptest.NewRequest(http.MethodGet, "http://resources/123/subscriptions?offset=1", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.valid.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.SubscriptionLoadSliceByteSubscription(
							infraTest.Load("handler.index.inputSubscription.json"),
						),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.index.inputResource.json"),
						),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientSubscriptionOptions(tt.subscriptionOptions...),
						testRepository.ClientResourceOptions(tt.resourceOptions...),
					)
					service, err := NewHandler(
						HandlerSubscriptionRepository(repo.Subscription()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string { return "123" }),
						HandlerGetSubscriptionID(func(r *http.Request) string { return "" }),
						HandlerGetSubscriptionURI(func(reId, subId string) string { return "" }),
						HandlerParsePagination(infraHTTP.ParsePagination(30)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)
					test.Runner(tt.status, tt.header, service.Index, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerShow(t *testing.T) {
	Convey("Feature: Serve a HTTP request to display a given subscription", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title               string
				req                 *http.Request
				status              int
				header              http.Header
				body                []byte
				subscriptionOptions []func(*testRepository.Subscription)
				resourceOptions     []func(*testRepository.Resource)
			}{
				{
					"return a not found",
					httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.notFound.json"),
					nil,
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.repositoryError.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionError(errors.New("generic error")),
					},
					nil,
				},
				{
					"return a subscription",
					httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.valid.output.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.SubscriptionLoadSliceByteSubscription(
							infraTest.Load("handler.show.valid.input.json"),
						),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.index.inputResource.json"),
						),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientSubscriptionOptions(tt.subscriptionOptions...),
						testRepository.ClientResourceOptions(tt.resourceOptions...),
					)
					service, err := NewHandler(
						HandlerSubscriptionRepository(repo.Subscription()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string { return "123" }),
						HandlerGetSubscriptionID(func(r *http.Request) string { return "456" }),
						HandlerGetSubscriptionURI(func(reId, subId string) string {
							return fmt.Sprintf("http://resources/%s/subscriptions/%s", reId, subId)
						}),
						HandlerParsePagination(infraHTTP.ParsePagination(30)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)
					test.Runner(tt.status, tt.header, service.Show, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerDelete(t *testing.T) {
	Convey("Feature: Serve a HTTP request to delete a given subscription", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title               string
				req                 *http.Request
				status              int
				header              http.Header
				body                []byte
				subscriptionOptions []func(*testRepository.Subscription)
				resourceOptions     []func(*testRepository.Resource)
			}{
				{
					"return a not found",
					httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.notFound.json"),
					nil,
					nil,
				},
				{
					"delete the subscription",
					httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
					http.StatusNoContent,
					http.Header{},
					nil,
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.SubscriptionLoadSliceByteSubscription(
							infraTest.Load("handler.show.valid.input.json"),
						),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.index.inputResource.json"),
						),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientSubscriptionOptions(tt.subscriptionOptions...),
						testRepository.ClientResourceOptions(tt.resourceOptions...),
					)
					service, err := NewHandler(
						HandlerSubscriptionRepository(repo.Subscription()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string { return "123" }),
						HandlerGetSubscriptionID(func(r *http.Request) string { return "456" }),
						HandlerGetSubscriptionURI(func(reId, subId string) string { return "" }),
						HandlerParsePagination(infraHTTP.ParsePagination(30)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)
					test.Runner(tt.status, tt.header, service.Delete, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerCreate(t *testing.T) {
	Convey("Feature: Serve a HTTP request to create a subscription", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title               string
				req                 *http.Request
				status              int
				header              http.Header
				body                []byte
				subscriptionOptions []func(*testRepository.Subscription)
				resourceOptions     []func(*testRepository.Resource)
			}{
				{
					"return a error because of a invalid body",
					httptest.NewRequest(http.MethodPost, "http://resources/123/subscriptions", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalid.1.json"),
					nil,
					nil,
				},
				{
					"return a error because of a invalid content (1)",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBufferString("{}"),
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalid.2.json"),
					nil,
					nil,
				},
				{
					"return a error because of a invalid content (2)",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.invalid.input.json"),
						),
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalid.3.json"),
					nil,
					nil,
				},
				{
					"return a error because the resource don't exist",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.input.json"),
						),
					),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalid.4.json"),
					nil,
					nil,
				},
				{
					"return a resource repository error",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.input.json"),
						),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalid.5.json"),
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error at repository")),
					},
				},
				{
					"return a subscription repository error",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.input.json"),
						),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.subscriptionRepositoryError.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionError(errors.New("error at repository")),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.create.resourceInput.json"),
						),
					},
				},
				{
					"return a subscription conflict",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.input.json"),
						),
					),
					http.StatusConflict,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.subscriptionRepositoryConflict.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionCreateId("456"),
						testRepository.SubscriptionLoadSliceByteSubscription(
							infraTest.Load("handler.create.inputArray.json"),
						),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.create.resourceInput.json"),
						),
					},
				},
				{
					"create the subscription",
					httptest.NewRequest(
						http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
							infraTest.Load("handler.create.input.json"),
						),
					),
					http.StatusCreated,
					http.Header{
						"Content-Type": []string{"application/json"},
						"Location":     []string{"http://resources/123/subscriptions/456"},
					},
					infraTest.Load("handler.create.create.json"),
					[]func(*testRepository.Subscription){
						testRepository.SubscriptionCreateId("456"),
						testRepository.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(
							infraTest.Load("handler.create.resourceInput.json"),
						),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientSubscriptionOptions(tt.subscriptionOptions...),
						testRepository.ClientResourceOptions(tt.resourceOptions...),
					)
					service, err := NewHandler(
						HandlerSubscriptionRepository(repo.Subscription()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string { return "123" }),
						HandlerGetSubscriptionID(func(r *http.Request) string { return "456" }),
						HandlerGetSubscriptionURI(func(reId, subId string) string {
							return fmt.Sprintf("http://resources/%s/subscriptions/%s", reId, subId)
						}),
						HandlerParsePagination(infraHTTP.ParsePagination(30)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)
					test.Runner(tt.status, tt.header, service.Create, tt.req, tt.body)
				})
			}
		})
	})
}
