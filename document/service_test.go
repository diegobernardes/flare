// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

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
	"github.com/diegobernardes/flare/infra/http/test"
	infraTest "github.com/diegobernardes/flare/infra/test"
	repoTest "github.com/diegobernardes/flare/repository/test"
)

func TestNewService(t *testing.T) {
	Convey("Given a list of valid service options", t, func() {
		writer, err := infraHTTP.NewWriter(log.NewNopLogger())
		So(err, ShouldBeNil)

		tests := [][]func(*Service){
			{
				ServiceDocumentRepository(repoTest.NewDocument()),
				ServiceResourceRepository(repoTest.NewResource()),
				ServiceGetDocumentId(func(*http.Request) string { return "" }),
				ServicePusher(newPushMock(nil)),
				ServiceWriter(writer),
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
				ServiceDocumentRepository(repoTest.NewDocument()),
			},
			{
				ServiceDocumentRepository(repoTest.NewDocument()),
				ServiceResourceRepository(repoTest.NewResource()),
			},
			{
				ServiceDocumentRepository(repoTest.NewDocument()),
				ServiceResourceRepository(repoTest.NewResource()),
				ServiceGetDocumentId(func(*http.Request) string { return "" }),
			},
			{
				ServiceDocumentRepository(repoTest.NewDocument()),
				ServiceResourceRepository(repoTest.NewResource()),
				ServiceGetDocumentId(func(*http.Request) string { return "" }),
				ServicePusher(newPushMock(nil)),
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

func TestServiceHandleShow(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository flare.DocumentRepositorier
		}{
			{
				"Not found",
				httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
				http.StatusNotFound,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleShow.notFound.json"),
				repoTest.NewDocument(),
			},
			{
				"Error at repository",
				httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleShow.errorRepository.json"),
				repoTest.NewDocument(repoTest.DocumentError(errors.New("error at repository"))),
			},
			{
				"Found",
				httptest.NewRequest(http.MethodGet, "http://documents/456", nil),
				http.StatusOK,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleShow.found.json"),
				repoTest.NewDocument(
					repoTest.DocumentLoadSliceByteDocument(infraTest.Load("serviceHandleShow.input.json")),
					repoTest.DocumentDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceDocumentRepository(tt.repository),
					ServiceResourceRepository(repoTest.NewResource()),
					ServiceGetDocumentId(func(r *http.Request) string {
						return strings.Replace(r.URL.Path, "/", "", -1)
					}),
					ServicePusher(newPushMock(nil)),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)
				test.Runner(tt.status, tt.header, service.HandleShow, tt.req, tt.body)
			})
		}
	})
}

func TestServiceHandleUpdate(t *testing.T) {
	Convey("Given a list of requests", t, func() {
		tests := []struct {
			title      string
			req        *http.Request
			status     int
			header     http.Header
			body       []byte
			repository flare.DocumentRepositorier
			pusher     pusher
		}{
			{
				"The request should be invalid because it has a query string",
				httptest.NewRequest(
					http.MethodPut, "http://documents/http://app.com/123?key=value", nil,
				),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleUpdate.invalid.1.json"),
				repoTest.NewDocument(),
				newPushMock(nil),
			},
			{
				"The request should be invalid because it has no body",
				httptest.NewRequest(
					http.MethodPut, "http://documents/http://app.com/123", nil,
				),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleUpdate.invalid.2.json"),
				repoTest.NewDocument(),
				newPushMock(nil),
			},
			{
				"The response should be a worker push error",
				httptest.NewRequest(
					http.MethodPut,
					"http://documents/http://app.com/123",
					bytes.NewBuffer(infraTest.Load("serviceHandleUpdate.input.json")),
				),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleUpdate.invalid.3.json"),
				repoTest.NewDocument(),
				newPushMock(errors.New("error during push")),
			},
			{
				"The response should be a valid response",
				httptest.NewRequest(
					http.MethodPut,
					"http://documents/http://app.com/123",
					bytes.NewBuffer(infraTest.Load("serviceHandleUpdate.input.json")),
				),
				http.StatusAccepted,
				http.Header{},
				nil,
				repoTest.NewDocument(),
				newPushMock(nil),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceDocumentRepository(tt.repository),
					ServiceResourceRepository(repoTest.NewResource()),
					ServiceGetDocumentId(func(r *http.Request) string { return "123" }),
					ServicePusher(tt.pusher),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)

				test.Runner(tt.status, tt.header, service.HandleUpdate, tt.req, tt.body)
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
			repository flare.DocumentRepositorier
			pusher     pusher
		}{
			{
				"The request should be invalid because it has a query string",
				httptest.NewRequest(
					http.MethodDelete, "http://documents/http://app.com/123?key=value", nil,
				),
				http.StatusBadRequest,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleDelete.invalid.1.json"),
				repoTest.NewDocument(),
				newPushMock(nil),
			},
			{
				"The response should be a worker push error",
				httptest.NewRequest(http.MethodDelete, "http://documents/http://app.com/123", nil),
				http.StatusInternalServerError,
				http.Header{"Content-Type": []string{"application/json"}},
				infraTest.Load("serviceHandleDelete.invalid.2.json"),
				repoTest.NewDocument(),
				newPushMock(errors.New("error during push")),
			},
			{
				"The response should be a valid response",
				httptest.NewRequest(http.MethodDelete, "http://documents/http://app.com/123", nil),
				http.StatusAccepted,
				http.Header{},
				nil,
				repoTest.NewDocument(),
				newPushMock(nil),
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				writer, err := infraHTTP.NewWriter(log.NewNopLogger())
				So(err, ShouldBeNil)

				service, err := NewService(
					ServiceDocumentRepository(tt.repository),
					ServiceResourceRepository(repoTest.NewResource()),
					ServiceGetDocumentId(func(r *http.Request) string { return "123" }),
					ServicePusher(tt.pusher),
					ServiceWriter(writer),
				)
				So(err, ShouldBeNil)

				test.Runner(tt.status, tt.header, service.HandleDelete, tt.req, tt.body)
			})
		}
	})
}
