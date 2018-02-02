// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
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
					HandlerRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetResourceURI(func(string) string { return "" }),
					HandlerParsePagination(infraHTTP.ParsePagination(0)),
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
					HandlerRepository(&memory.Resource{}),
				},
				{
					HandlerRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
				},
				{
					HandlerRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetResourceURI(func(string) string { return "" }),
				},
				{
					HandlerRepository(&memory.Resource{}),
					HandlerGetResourceID(func(*http.Request) string { return "" }),
					HandlerGetResourceURI(func(string) string { return "" }),
					HandlerParsePagination(infraHTTP.ParsePagination(0)),
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
	Convey("Feature: Serve a HTTP request to display a list of resources", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title             string
				req               *http.Request
				status            int
				header            http.Header
				body              []byte
				repositoryOptions []func(*testRepository.Resource)
			}{
				{
					"return a pagination error because of a invalid limit (1)",
					httptest.NewRequest(http.MethodGet, "http://resources?limit=sample", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.1.json"),
					nil,
				},
				{
					"return a pagination error because of a invalid limit (2)",
					httptest.NewRequest(http.MethodGet, "http://resources?limit=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.2.json"),
					nil,
				},
				{
					"return a pagination error because of a invalid offset",
					httptest.NewRequest(http.MethodGet, "http://resources?offset=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.3.json"),
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest(http.MethodGet, "http://resources", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.repositoryError.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error during repository search")),
					},
				},
				{
					"return a list of resources (1)",
					httptest.NewRequest(http.MethodGet, "http://resources", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.listResources.1.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.1.json")),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
				{
					"return a list of resources (2)",
					httptest.NewRequest(http.MethodGet, "http://resources?limit=10", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.listResources.2.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.2.json")),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
				{
					"return a list of resources (3)",
					httptest.NewRequest(http.MethodGet, "http://resources?limit=10&offset=1", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.listResources.3.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.2.json")),
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)
					service, err := NewHandler(
						HandlerRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string {
							return strings.Replace(r.URL.String(), "http://resources/", "", -1)
						}),
						HandlerGetResourceURI(func(string) string { return "" }),
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
	Convey("Feature: Serve a HTTP request to display a given resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title             string
				req               *http.Request
				status            int
				header            http.Header
				body              []byte
				repositoryOptions []func(*testRepository.Resource)
			}{
				{
					"return a not found",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.notFound.json"),
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.repositoryError.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error during repository search")),
					},
				},
				{
					"return a resource",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.resource.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.3.json")),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)
					service, err := NewHandler(
						HandlerRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string {
							return strings.Replace(r.URL.String(), "http://resources/", "", -1)
						}),
						HandlerGetResourceURI(func(string) string { return "" }),
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
	Convey("Feature: Serve a HTTP request to delete a given resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title             string
				req               *http.Request
				status            int
				header            http.Header
				body              []byte
				repositoryOptions []func(*testRepository.Resource)
			}{
				{
					"return a not found",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.notFound.json"),
					nil,
				},
				{
					"return a repository error",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.repositoryError.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error during repository delete")),
					},
				},
				{
					"delete the resource",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusNoContent,
					http.Header{},
					nil,
					[]func(*testRepository.Resource){
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.3.json")),
					},
				},
			}

			for _, tt := range tests {
				Convey(tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)
					service, err := NewHandler(
						HandlerRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string {
							return strings.Replace(r.URL.String(), "http://resources/", "", -1)
						}),
						HandlerGetResourceURI(func(string) string { return "" }),
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
	Convey("Feature: Serve a HTTP request to create a resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title             string
				req               *http.Request
				status            int
				header            http.Header
				body              []byte
				repositoryOptions []func(*testRepository.Resource)
			}{
				{
					"return a error during body parse",
					httptest.NewRequest(http.MethodPost, "http://resources/123", bytes.NewBuffer([]byte{})),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidBody.json"),
					nil,
				},
				{
					"return a invalid resource",
					httptest.NewRequest(http.MethodPost, "http://resources/123", bytes.NewBufferString("{}")),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidResource.json"),
					nil,
				},
				{
					"return a resource conflict error",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources/123",
						bytes.NewBuffer(infraTest.Load("handler.create.input.json")),
					),
					http.StatusConflict,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.conflict.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.ResourceLoadSliceByteResource(infraTest.Load("resource.input.3.json")),
					},
				},
				{
					"return a repository error",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources/123",
						bytes.NewBuffer(infraTest.Load("handler.create.input.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.repositoryError.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceError(errors.New("error during repository save")),
					},
				},
				{
					"create the document",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources/123",
						bytes.NewBuffer(infraTest.Load("handler.create.input.json")),
					),
					http.StatusCreated,
					http.Header{
						"Content-Type": []string{"application/json"},
						"Location":     []string{"http://resources/123"},
					},
					infraTest.Load("handler.create.resource.json"),
					[]func(*testRepository.Resource){
						testRepository.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						testRepository.ResourceCreateID("123"),
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					repo := testRepository.NewClient(
						testRepository.ClientResourceOptions(tt.repositoryOptions...),
					)
					service, err := NewHandler(
						HandlerRepository(repo.Resource()),
						HandlerGetResourceID(func(r *http.Request) string {
							return strings.Replace(r.URL.String(), "http://resources/", "", -1)
						}),
						HandlerGetResourceURI(func(id string) string {
							return "http://resources/" + id
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
