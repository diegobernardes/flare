// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

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

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/repository/memory"
)

func TestNewService(t *testing.T) {
	Convey("Given a list of valid service options", t, func() {
		tests := [][]func(*Service){
			{
				ServiceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetResourceURI(func(string) string { return "" }),
				ServiceParsePagination(infraHTTP.ParsePagination(0)),
				ServiceWriteResponse(infraHTTP.WriteResponse(log.NewNopLogger())),
			},
		}

		Convey("The service initialization should not return error", func() {
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
				ServiceRepository(memory.NewResource()),
			},
			{
				ServiceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
			},
			{
				ServiceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetResourceURI(func(string) string { return "" }),
			},
			{
				ServiceRepository(memory.NewResource()),
				ServiceGetResourceID(func(*http.Request) string { return "" }),
				ServiceGetResourceURI(func(string) string { return "" }),
				ServiceParsePagination(infraHTTP.ParsePagination(0)),
			},
		}

		Convey("The service initialization should return error", func() {
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
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository resourceRepository
		}{
			{
				"The request should have a invalid pagination 1",
				httptest.NewRequest("GET", "http://resources?limit=sample", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.invalidPagination.1.json"),
				resourceRepository{},
			},
			{
				"The request should have a invalid pagination 2",
				httptest.NewRequest("GET", "http://resources?limit=-1", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.invalidPagination.2.json"),
				resourceRepository{},
			},
			{
				"The request should have a invalid pagination 3",
				httptest.NewRequest("GET", "http://resources?offset=-1", nil),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.invalidPagination.3.json"),
				resourceRepository{},
			},
			{
				"The response should be a repository error",
				httptest.NewRequest("GET", "http://resources", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.repositoryError.json"),
				resourceRepository{err: errors.New("error during repository search")},
			},
			{
				"The response should be a valid search 1",
				httptest.NewRequest("GET", "http://resources", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.validSearch.1.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "1",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
			{
				"The response should be a valid search 2",
				httptest.NewRequest("GET", "http://resources?limit=10", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.validSearch.2.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "1",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
						{
							ID:        "2",
							Addresses: []string{"http://app2.com", "https://app2.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
			{
				"The response should be a valid search 3",
				httptest.NewRequest("GET", "http://resources?limit=10&offset=1", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleIndex.validSearch.3.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "1",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
						{
							ID:        "2",
							Addresses: []string{"http://app2.com", "https://app2.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				service, err := NewService(
					ServiceRepository(&tt.repository),
					ServiceGetResourceID(func(r *http.Request) string {
						return strings.Replace(r.URL.String(), "http://resources/", "", -1)
					}),
					ServiceGetResourceURI(func(string) string { return "" }),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriteResponse(infraHTTP.WriteResponse(log.NewNopLogger())),
				)
				if err != nil {
					t.Error(errors.Wrap(err, "error during service initialization"))
					t.FailNow()
				}

				httpRunner(tt.status, tt.header, service.HandleIndex, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleShow(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository resourceRepository
		}{
			{
				"The response should be a resource not found",
				httptest.NewRequest("GET", "http://resources/123", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleShow.notFound.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{},
					"",
				),
			},
			{
				"The response should be a error during search",
				httptest.NewRequest("GET", "http://resources/123", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleShow.repositoryError.json"),
				resourceRepository{err: errors.New("error during repository search")},
			},
			{
				"The response should be a resource",
				httptest.NewRequest("GET", "http://resources/123", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleShow.valid.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "123",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				service, err := NewService(
					ServiceRepository(&tt.repository),
					ServiceGetResourceID(func(r *http.Request) string {
						return strings.Replace(r.URL.String(), "http://resources/", "", -1)
					}),
					ServiceGetResourceURI(func(string) string { return "" }),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriteResponse(infraHTTP.WriteResponse(log.NewNopLogger())),
				)
				if err != nil {
					t.Error(errors.Wrap(err, "error during service initialization"))
					t.FailNow()
				}

				httpRunner(tt.status, tt.header, service.HandleShow, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleDelete(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository resourceRepository
		}{
			{
				"The response should be a resource not found",
				httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleDelete.notFound.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{},
					"",
				),
			},
			{
				"The response should be a error during search",
				httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleDelete.repositoryError.json"),
				resourceRepository{err: errors.New("error during repository delete")},
			},
			{
				"The response should be the result of a deleted resource",
				httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
				http.StatusNoContent,
				http.Header{},
				nil,
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "123",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				service, err := NewService(
					ServiceRepository(&tt.repository),
					ServiceGetResourceID(func(r *http.Request) string {
						return strings.Replace(r.URL.String(), "http://resources/", "", -1)
					}),
					ServiceGetResourceURI(func(string) string { return "" }),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriteResponse(infraHTTP.WriteResponse(log.NewNopLogger())),
				)
				if err != nil {
					t.Error(errors.Wrap(err, "error during service initialization"))
					t.FailNow()
				}

				httpRunner(tt.status, tt.header, service.HandleDelete, tt.req, tt.body)
			})
		}
	})
}
func TestServiceHandleCreate(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository resourceRepository
		}{
			{
				"The request should have a invalid resource 1",
				httptest.NewRequest(http.MethodPost, "http://resources/123", bytes.NewBuffer([]byte{})),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleCreate.invalid.1.json"),
				resourceRepository{},
			},
			{
				"The request should have a invalid pagination 2",
				httptest.NewRequest(http.MethodPost, "http://resources/123", bytes.NewBufferString("{}")),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleCreate.invalid.2.json"),
				resourceRepository{},
			},
			{
				"The request should be a resource conflict",
				httptest.NewRequest(
					http.MethodPost,
					"http://resources/123",
					bytes.NewBuffer(load("serviceHandleCreate.input.1.json")),
				),
				http.StatusConflict,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleCreate.conflict.1.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{
						{
							ID:        "123",
							Addresses: []string{"http://app1.com", "https://app1.io"},
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Path:      "/resources/{*}",
							Change: flare.ResourceChange{
								Field:      "updatedAt",
								Kind:       flare.ResourceChangeDate,
								DateFormat: "2006-01-02T15:04:05Z07:00",
							},
						},
					},
					"",
				),
			},
			{
				"The response should be a repository error",
				httptest.NewRequest(
					http.MethodPost,
					"http://resources/123",
					bytes.NewBuffer(load("serviceHandleCreate.input.1.json")),
				),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				load("serviceHandleCreate.repositoryError.json"),
				resourceRepository{err: errors.New("error during repository save")},
			},
			{
				"The response should be the result of a created resource",
				httptest.NewRequest(
					http.MethodPost,
					"http://resources/123",
					bytes.NewBuffer(load("serviceHandleCreate.input.1.json")),
				),
				http.StatusCreated,
				http.Header{
					"Content-Type": []string{"application/json"},
					"Location":     []string{"http://resources/123"},
				},
				load("serviceHandleCreate.valid.json"),
				newResourceRepository(
					time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					[]flare.Resource{},
					"123",
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				service, err := NewService(
					ServiceRepository(&tt.repository),
					ServiceGetResourceID(func(r *http.Request) string {
						return strings.Replace(r.URL.String(), "http://resources/", "", -1)
					}),
					ServiceGetResourceURI(func(id string) string {
						return "http://resources/" + id
					}),
					ServiceParsePagination(infraHTTP.ParsePagination(30)),
					ServiceWriteResponse(infraHTTP.WriteResponse(log.NewNopLogger())),
				)
				if err != nil {
					t.Error(errors.Wrap(err, "error during service initialization"))
					t.FailNow()
				}

				httpRunner(tt.status, tt.header, service.HandleCreate, tt.req, tt.body)
			})
		}
	})
}
