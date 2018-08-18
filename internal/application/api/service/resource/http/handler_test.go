//go:generate moq -out http_mock_test.go . service serviceError
package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
	"github.com/diegobernardes/flare/internal/application/api/infra/http/test"
	infraTest "github.com/diegobernardes/flare/internal/infra/test"
)

func TestHandlerInit(t *testing.T) {
	Convey("Feature: Handler initialization", t, func() {
		Convey("Given a list handlers", func() {
			writer, err := infraHTTP.NewWriter(log.NewNopLogger())
			So(err, ShouldBeNil)

			tests := []struct {
				title       string
				handler     Handler
				shouldError bool
			}{
				{
					"initialize the handler",
					Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &serviceMock{},
					},
					false,
				},
				{
					"have a error because of missing parse pagination",
					Handler{
						ExtractID: func(r *http.Request) string { return "" },
						GenURI:    func(string) string { return "" },
						Writer:    writer,
						Service:   &serviceMock{},
					},
					true,
				},
				{
					"have a error because of missing writer",
					Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &serviceMock{},
					},
					true,
				},
				{
					"have a error because of missing service",
					Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
					},
					true,
				},
				{
					"have a error because of missing extract id",
					Handler{
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &serviceMock{},
					},
					true,
				},
				{
					"have a error because of missing gen uri",
					Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &serviceMock{},
					},
					true,
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					err := tt.handler.Init()
					if tt.shouldError {
						So(err, ShouldNotBeNil)
					} else {
						So(err, ShouldBeNil)
					}
				})
			}
		})
	})
}

func TestHandlerIndex(t *testing.T) {
	Convey("Feature: Serve a HTTP request to display a list of resources", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title   string
				req     *http.Request
				status  int
				header  http.Header
				body    []byte
				service serviceMock
			}{
				{
					"return a invalid query error",
					httptest.NewRequest(http.MethodGet, "http://resources?not-valid=query", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidQuery.json"),
					serviceMock{},
				},
				{
					"return a pagination error because of a invalid limit",
					httptest.NewRequest(http.MethodGet, "http://resources?limit=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.1.json"),
					serviceMock{},
				},
				{
					"return a pagination error because of a invalid offset",
					httptest.NewRequest(http.MethodGet, "http://resources?offset=-1", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.invalidPagination.2.json"),
					serviceMock{},
				},
				{
					"return a client service error",
					httptest.NewRequest(http.MethodGet, "http://resources", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.clientServiceError.json"),
					serviceMock{
						FindFunc: func(
							ctx context.Context, pagination infraHTTP.Pagination,
						) ([]internal.Resource, infraHTTP.Pagination, error) {
							return nil, pagination, genError("client")
						},
					},
				},
				{
					"return a server service error",
					httptest.NewRequest(http.MethodGet, "http://resources", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.internalServiceError.json"),
					serviceMock{
						FindFunc: func(
							ctx context.Context, pagination infraHTTP.Pagination,
						) ([]internal.Resource, infraHTTP.Pagination, error) {
							return nil, pagination, genError("server")
						},
					},
				},
				{
					"return a list of resources",
					httptest.NewRequest(http.MethodGet, "http://resources", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.index.listResources.json"),
					serviceMock{
						FindFunc: func(
							ctx context.Context, pagination infraHTTP.Pagination,
						) ([]internal.Resource, infraHTTP.Pagination, error) {
							payload := infraTest.Load("handler.index.listResources.json")
							result, resultPagination := loadIndexResponse(payload)
							return result, resultPagination, nil
						},
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					handler := Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &tt.service,
					}
					So(handler.Init(), ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Index, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerShow(t *testing.T) {
	Convey("Feature: Serve a HTTP request to display a given resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title   string
				req     *http.Request
				status  int
				header  http.Header
				body    []byte
				service serviceMock
			}{
				{
					"return a invalid query error",
					httptest.NewRequest(http.MethodGet, "http://resources/123?not-valid=query", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.invalidQuery.json"),
					serviceMock{},
				},
				{
					"return a not found service error",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusNotFound,
					http.Header{},
					nil,
					serviceMock{
						FindByIDFunc: func(ctx context.Context, id string) (*internal.Resource, error) {
							return nil, nil
						},
					},
				},
				{
					"return a generic service error",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.genericServiceError.json"),
					serviceMock{
						FindByIDFunc: func(ctx context.Context, id string) (*internal.Resource, error) {
							return nil, genError("")
						},
					},
				},
				{
					"return a resource",
					httptest.NewRequest(http.MethodGet, "http://resources/123", nil),
					http.StatusOK,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.show.success.json"),
					serviceMock{
						FindByIDFunc: func(context.Context, string) (*internal.Resource, error) {
							r := infraTest.LoadResource(infraTest.Load("handler.show.success.json"))
							return &r, nil
						},
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					handler := Handler{
						ExtractID:       func(r *http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &tt.service,
					}
					So(handler.Init(), ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Show, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerCreate(t *testing.T) {
	Convey("Feature: Serve a HTTP request to create a resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title   string
				req     *http.Request
				status  int
				header  http.Header
				body    []byte
				service serviceMock
			}{
				{
					"return a invalid query error",
					httptest.NewRequest(http.MethodPost, "http://resources?not-valid=query", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidQuery.json"),
					serviceMock{},
				},
				{
					"return a error because a empty body",
					httptest.NewRequest(http.MethodPost, "http://resources", bytes.NewBuffer([]byte{})),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidBody.1.json"),
					serviceMock{},
				},
				{
					"return a error because a unknow json field",
					httptest.NewRequest(
						http.MethodPost, "http://resources", bytes.NewBufferString(`{"field": "unknow"}`)),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidBody.2.json"),
					serviceMock{},
				},
				{
					"return a error because of a invalid endpoint",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources",
						bytes.NewBuffer(infraTest.Load("handler.create.input.1.json")),
					),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.invalidBody.3.json"),
					serviceMock{},
				},
				{
					"return a service error",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources",
						bytes.NewBuffer(infraTest.Load("handler.create.input.2.json")),
					),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.serviceError.json"),
					serviceMock{
						CreateFunc: func(ctx context.Context, resource internal.Resource) (string, error) {
							return "", genError("")
						},
					},
				},
				{
					"return a service conflict error",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources",
						bytes.NewBuffer(infraTest.Load("handler.create.input.2.json")),
					),
					http.StatusConflict,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.create.serviceConflictError.json"),
					serviceMock{
						CreateFunc: func(context.Context, internal.Resource) (string, error) {
							return "", genError("alreadyExists")
						},
					},
				},
				{
					"create the resource",
					httptest.NewRequest(
						http.MethodPost,
						"http://resources",
						bytes.NewBuffer(infraTest.Load("handler.create.input.2.json")),
					),
					http.StatusCreated,
					http.Header{
						"Content-Type": []string{"application/json"},
						"Location":     []string{"http://resources/123"},
					},
					infraTest.Load("handler.create.success.json"),
					serviceMock{
						CreateFunc: func(context.Context, internal.Resource) (string, error) {
							return "123", nil
						},
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					handler := Handler{
						ExtractID: func(r *http.Request) string { return "" },
						GenURI: func(id string) string {
							return fmt.Sprintf("http://resources/%s", id)
						},
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &tt.service,
					}
					So(handler.Init(), ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Create, tt.req, tt.body)
				})
			}
		})
	})
}

func TestHandlerDelete(t *testing.T) {
	Convey("Feature: Serve a HTTP request to delete a given resource", t, func() {
		Convey("Given a list of requests", func() {
			tests := []struct {
				title   string
				req     *http.Request
				status  int
				header  http.Header
				body    []byte
				service serviceMock
			}{
				{
					"return a invalid query error",
					httptest.NewRequest(http.MethodDelete, "http://resources?not-valid=query", nil),
					http.StatusBadRequest,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.invalidQuery.json"),
					serviceMock{},
				},
				{
					"return a service error",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusInternalServerError,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.serviceError.json"),
					serviceMock{
						DeleteFunc: func(context.Context, string) error { return genError("") },
					},
				},
				{
					"return a service not found error",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusNotFound,
					http.Header{"Content-Type": []string{"application/json"}},
					infraTest.Load("handler.delete.serviceNotFoundError.json"),
					serviceMock{
						DeleteFunc: func(context.Context, string) error { return genError("notFound") },
					},
				},
				{
					"delete the resource",
					httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
					http.StatusNoContent,
					http.Header{},
					nil,
					serviceMock{
						DeleteFunc: func(context.Context, string) error { return nil },
					},
				},
			}

			for _, tt := range tests {
				Convey("Should "+tt.title, func() {
					writer, err := infraHTTP.NewWriter(log.NewNopLogger())
					So(err, ShouldBeNil)

					handler := Handler{
						ExtractID:       func(*http.Request) string { return "" },
						GenURI:          func(string) string { return "" },
						Writer:          writer,
						ParsePagination: infraHTTP.ParsePagination(30),
						Service:         &tt.service,
					}
					So(handler.Init(), ShouldBeNil)

					test.Runner(tt.status, tt.header, handler.Delete, tt.req, tt.body)
				})
			}
		})
	})
}
