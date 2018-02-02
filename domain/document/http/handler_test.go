// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	subscriptionTest "github.com/diegobernardes/flare/domain/subscription/worker/test"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/http/test"
	infraTest "github.com/diegobernardes/flare/infra/test"
	testRepository "github.com/diegobernardes/flare/provider/test/repository"
)

func TestNewHandler(t *testing.T) {
	Convey("Feature: Create a new instance of Handler", t, func() {
		Convey("Given a list of valid parameters", func() {
			writer, err := infraHTTP.NewWriter(log.NewNopLogger())
			So(err, ShouldBeNil)

			options := []func(*Handler){
				HandlerDocumentRepository(&testRepository.Document{}),
				HandlerResourceRepository(&testRepository.Resource{}),
				HandlerGetDocumentID(func(*http.Request) string { return "" }),
				HandlerSubscriptionTrigger(&subscriptionTest.Trigger{}),
				HandlerWriter(writer),
			}

			Convey("Should output a valid Handler", func() {
				_, err := NewHandler(options...)
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a list of invalid parameters", func() {
			tests := [][]func(*Handler){
				{},
				{
					HandlerDocumentRepository(&testRepository.Document{}),
				},
				{
					HandlerDocumentRepository(&testRepository.Document{}),
					HandlerResourceRepository(&testRepository.Resource{}),
				},
				{
					HandlerDocumentRepository(&testRepository.Document{}),
					HandlerResourceRepository(&testRepository.Resource{}),
					HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
				},
				{
					HandlerDocumentRepository(&testRepository.Document{}),
					HandlerResourceRepository(&testRepository.Resource{}),
					HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
					HandlerGetDocumentID(func(*http.Request) string { return "" }),
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

func TestServiceHandlerShow(t *testing.T) {
	Convey("Feature: Serve a HTTP request to display a given document", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title   string
				req     *http.Request
				status  int
				header  http.Header
				body    []byte
				options []func(*testRepository.Document)
			}{
				{
					"return a not found",
					httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.notFound.json"),
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.repositoryError.json"),
					[]func(*testRepository.Document){
						testRepository.DocumentError(errors.New("error at repository")),
					},
				},
				{
					"return a document",
					httptest.NewRequest(http.MethodGet, "http://documents/456", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.found.json"),
					[]func(*testRepository.Document){
						testRepository.DocumentLoadSliceByteDocument(infraTest.Load("document.1.json")),
						testRepository.DocumentDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(testRepository.ClientDocumentOptions(tt.options...))
					handler, err := NewHandler(
						HandlerDocumentRepository(repo.Document()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetDocumentID(func(r *http.Request) string {
							return strings.Replace(r.URL.Path, "/", "", -1)
						}),
						HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)
					test.Runner(tt.status, tt.header, handler.Show, tt.req, tt.body)
				})
			}
		})
	})
}

func TestServiceHandlerUpdate(t *testing.T) {
	Convey("Feature: Serve a HTTP request to update a given document", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title           string
				req             *http.Request
				status          int
				header          http.Header
				body            []byte
				documentOptions []func(*testRepository.Document)
				resourceOptions []func(*testRepository.Resource)
				triggerError    error
			}{
				{
					"return a error because it has a query string",
					httptest.NewRequest(
						http.MethodPut, "http://documents/http://app.com/users/123?key=value", nil,
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.update.invalidQueryString.json"),
					nil,
					nil,
					nil,
				},
				{
					"return a error because of a repository document find error",
					httptest.NewRequest(
						http.MethodPut,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer(infraTest.Load("handler.update.input.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.update.documentFindError.json"),
					[]func(*testRepository.Document){
						testRepository.DocumentFindOneError(
							&testRepository.DocumentErr{Message: "generic error"},
						),
					},
					nil,
					nil,
				},
				{
					"return a error because of a document parse error",
					httptest.NewRequest(
						http.MethodPut,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer([]byte("{}")),
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.update.parseError.json"),
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					nil,
				},
				{
					"return a error because of a repository document update error",
					httptest.NewRequest(
						http.MethodPut,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer(infraTest.Load("handler.update.input.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.update.updateError.json"),
					[]func(*testRepository.Document){
						testRepository.DocumentUpdateError(errors.New("error during update")),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					nil,
				},
				{
					"return a error because of a subscription trigger error",
					httptest.NewRequest(
						http.MethodPut,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer(infraTest.Load("handler.update.input.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.update.triggerError.json"),
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					errors.New("error during push"),
				},
				{
					"update the document",
					httptest.NewRequest(
						http.MethodPut,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer(infraTest.Load("handler.update.input.json")),
					),
					http.StatusAccepted,
					http.Header{},
					nil,
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					nil,
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientDocumentOptions(tt.documentOptions...),
						testRepository.ClientResourceOptions(tt.resourceOptions...),
					)
					handler, err := NewHandler(
						HandlerDocumentRepository(repo.Document()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetDocumentID(func(r *http.Request) string { return "http://app.com/users/123" }),
						HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(tt.triggerError)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Update, tt.req, tt.body)
				})
			}
		})
	})
}

func TestServiceHandlerDelete(t *testing.T) {
	Convey("Feature: Serve a HTTP request to delete a given document", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title           string
				req             *http.Request
				status          int
				header          http.Header
				body            []byte
				documentOptions []func(*testRepository.Document)
				resourceOptions []func(*testRepository.Resource)
				triggerError    error
			}{
				{
					"return a error because it has a query string",
					httptest.NewRequest(
						http.MethodDelete, "http://documents/http://app.com/users/123?key=value", nil,
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.invalidQueryString.json"),
					nil,
					nil,
					nil,
				},
				{
					"return a error because of a repository document find error",
					httptest.NewRequest(
						http.MethodDelete,
						"http://documents/http://app.com/users/123",
						bytes.NewBuffer(infraTest.Load("handler.update.input.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.documentFindError.json"),
					[]func(*testRepository.Document){
						testRepository.DocumentFindOneError(
							&testRepository.DocumentErr{Message: "generic error"},
						),
					},
					nil,
					nil,
				},
				{
					"return a error because of a subscription trigger error",
					httptest.NewRequest(http.MethodDelete, "http://documents/http://app1.com/users/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.triggerError.json"),
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					errors.New("error during push"),
				},
				{
					"delete the document",
					httptest.NewRequest(http.MethodDelete, "http://documents/http://app1.com/users/123", nil),
					http.StatusAccepted,
					http.Header{},
					nil,
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
					nil,
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.resourceOptions...),
						testRepository.ClientDocumentOptions(tt.documentOptions...),
					)
					handler, err := NewHandler(
						HandlerDocumentRepository(repo.Document()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetDocumentID(func(r *http.Request) string { return "http://app.com/users/123" }),
						HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(tt.triggerError)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Delete, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerFetchResource(t *testing.T) {
	Convey("Feature: Get a Resource from a document ID", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title           string
				status          int
				header          http.Header
				body            []byte
				documentID      string
				resource        *flare.Resource
				documentOptions []func(*testRepository.Document)
				resourceOptions []func(*testRepository.Resource)
			}{
				{
					"return a error because of a repository document find error",
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.fetchResource.documentError.json"),
					"http://app1.com/users/123",
					nil,
					[]func(*testRepository.Document){
						testRepository.DocumentFindOneError(
							&testRepository.DocumentErr{Message: "generic error"},
						),
					},
					nil,
				},
				{
					"return a error because of a repository resource not found error",
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.fetchResource.resourceNotFound.json"),
					"http://app1.com/users/123",
					nil,
					nil,
					nil,
				},
				{
					"return a error because of a repository resource find error",
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.fetchResource.resourceError.json"),
					"http://app1.com/users/123",
					nil,
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("generic error")),
					},
				},
				{
					"return the resource based on a document repository search",
					0,
					nil,
					nil,
					"456",
					nil,
					[]func(*testRepository.Document){
						testRepository.DocumentLoadSliceByteDocument(infraTest.Load("document.1.json")),
						testRepository.DocumentDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
				},
				{
					"return the resource based on a resource repository search",
					0,
					nil,
					nil,
					"http://app.com/users/123",
					nil,
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.1.json")),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.resourceOptions...),
						testRepository.ClientDocumentOptions(tt.documentOptions...),
					)
					handler, err := NewHandler(
						HandlerDocumentRepository(repo.Document()),
						HandlerResourceRepository(repo.Resource()),
						HandlerGetDocumentID(func(r *http.Request) string { return "http://app.com/users/123" }),
						HandlerSubscriptionTrigger(subscriptionTest.NewTrigger(nil)),
						HandlerWriter(writer),
					)
					So(err, ShouldBeNil)

					w := httptest.NewRecorder()
					resource := handler.fetchResource(context.Background(), tt.documentID, w)
					if resource == nil {
						So(w.Code, ShouldEqual, tt.status)
						So(w.HeaderMap, ShouldResemble, tt.header)
						infraTest.CompareJSONBytes(w.Body.Bytes(), tt.body)
					} else {
						So(*resource, ShouldResemble, *resource)
					}
				})
			}
		})
	})
}
