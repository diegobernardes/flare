// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
)

func TestPaginationMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal a pagination", t, func() {
		Convey("Given a list of paginations", func() {
			tests := []struct {
				pagination pagination
				expected   string
			}{
				{
					pagination{Limit: 30, Offset: 0},
					`{"limit":30,"offset":0,"total":0}`,
				},
				{
					pagination{Limit: 10, Offset: 30, Total: 120},
					`{"limit":10,"offset":30,"total":120}`,
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					content, err := tt.pagination.MarshalJSON()
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, tt.expected)
				}
			})
		})
	})
}

func TestResourceMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal a resource", t, func() {
		Convey("Given a list of resources", func() {
			tests := []struct {
				resource resource
				expected []byte
			}{
				{
					resource{
						ID:        "id",
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Addresses: []string{"http://flare.io", "https://flare.com"},
						Path:      "/resources/{*}",
						Change:    flare.ResourceChange{Field: "version"},
					},
					infraTest.Load("resource.1.json"),
				},
				{
					resource{
						ID:        "id",
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Addresses: []string{"http://flare.io", "https://flare.com"},
						Path:      "/resources/{*}",
						Change: flare.ResourceChange{
							Field:  "updatedAt",
							Format: "2006-01-02",
						},
					},
					infraTest.Load("resource.2.json"),
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					content, err := tt.resource.MarshalJSON()
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(content, tt.expected)
				}
			})
		})
	})
}

func TestResponseMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal a response", t, func() {
		Convey("Given a list of responses", func() {
			tests := []struct {
				response response
				expected []byte
			}{
				{
					response{
						Resource: &resource{
							ID:        "123",
							Addresses: []string{"http://address1", "https://address2"},
							Path:      "/products/{*}",
							CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
							Change:    flare.ResourceChange{Field: "version"},
						},
					},
					infraTest.Load("response.marshalJSON.1.json"),
				},
				{
					response{
						Pagination: (*pagination)(&flare.Pagination{Limit: 10, Total: 30, Offset: 20}),
						Resources: []resource{
							{
								ID:        "123",
								Addresses: []string{"http://address1", "https://address2"},
								Path:      "/products/{*}",
								CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
								Change:    flare.ResourceChange{Field: "version"},
							},
						},
					},
					infraTest.Load("response.marshalJSON.2.json"),
				},
			}

			Convey("Should output a valid JSON", func() {
				for _, tt := range tests {
					content, err := tt.response.MarshalJSON()
					So(err, ShouldBeNil)
					infraTest.CompareJSONBytes(content, tt.expected)
				}
			})
		})
	})
}

func TestResourceCreateValidAddresses(t *testing.T) {
	Convey("Feature: Validate resourceCreate.addresses", t, func() {
		Convey("Given a list of valid addresses", func() {
			tests := []resourceCreate{
				{Addresses: []string{"http://app.io"}},
				{Addresses: []string{"https://app.com"}},
				{Addresses: []string{"http://app.io", "https://app.com"}},
			}

			Convey("Should not output a error", func() {
				for _, tt := range tests {
					result := tt.validAddresses()
					So(result, ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid addresses", func() {
			tests := []resourceCreate{
				{},
				{Addresses: []string{""}},
				{Addresses: []string{"tcp://127.0.0.1:8080"}},
				{Addresses: []string{"%zzzzz"}},
				{Addresses: []string{"http://app,com/teste"}},
				{Addresses: []string{"http://app,com#fragment"}},
				{Addresses: []string{"http://app,com?project=flare"}},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					result := tt.validAddresses()
					So(result, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestResourceCreateValid(t *testing.T) {
	type resourceCreateChange struct {
		Field  string `json:"field"`
		Format string `json:"format"`
	}

	Convey("Feature: Validate resourceCreate", t, func() {
		Convey("Given a list of valid resourceCreate", func() {
			tests := []resourceCreate{
				{
					Addresses: []string{"http://app.com"},
					Path:      "/users/{*}",
					Change: resourceCreateChange{
						Field:  "updatedAt",
						Format: "2006-01-02T15:04:05Z07:00",
					},
				},
				{
					Addresses: []string{"http://app.com"},
					Path:      "/users/{*}",
					Change:    resourceCreateChange{Field: "sequence"},
				},
				{
					Addresses: []string{"http://app.com"},
					Path:      "/users/{*}/{id}",
					Change:    resourceCreateChange{Field: "sequence"},
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					result := tt.valid()
					So(result, ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid resourceCreate", func() {
			tests := []resourceCreate{
				{},
				{Addresses: []string{"http://app.com"}},
				{Addresses: []string{"http://app.com"}, Path: "users"},
				{Addresses: []string{"http://app.com"}, Path: "/users"},
				{Addresses: []string{"http://app.com"}, Path: "/users/"},
				{Addresses: []string{"http://app.com"}, Path: "/users/{*}-path/posts/{*}"},
				{Addresses: []string{"http://app.com"}, Path: "/users/{*}"},
				{Addresses: []string{"http://app.com"}, Change: resourceCreateChange{Field: "sequence"}},
			}

			Convey("Should return a error", func() {
				for _, tt := range tests {
					result := tt.valid()
					So(result, ShouldNotBeNil)
				}
			})
		})
	})
}
