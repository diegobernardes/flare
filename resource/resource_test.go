// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
	infraTest "github.com/diegobernardes/flare/infra/test"
)

func TestPaginationMarshalJSON(t *testing.T) {
	Convey("Given a list of valid paginations", t, func() {
		tests := []struct {
			input  pagination
			output string
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

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(content), ShouldEqual, tt.output)
			}
		})
	})
}

func TestResourceMarshalJSON(t *testing.T) {
	Convey("Given a list of valid resources", t, func() {
		tests := []struct {
			input  resource
			output []byte
		}{
			{
				resource{
					ID:        "id",
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Addresses: []string{"http://flare.io", "https://flare.com"},
					Path:      "/resources/{*}",
					Change: flare.ResourceChange{
						Field: "version",
						Kind:  flare.ResourceChangeInteger,
					},
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
						Field:      "updatedAt",
						Kind:       flare.ResourceChangeDate,
						DateFormat: "2006-01-02",
					},
				},
				infraTest.Load("resource.2.json"),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)

				c1, c2 := make(map[string]interface{}), make(map[string]interface{})
				err = json.Unmarshal([]byte(content), &c1)
				So(err, ShouldBeNil)

				err = json.Unmarshal([]byte(tt.output), &c2)
				So(err, ShouldBeNil)

				So(c1, ShouldResemble, c2)
			}
		})
	})
}

func TestResponseMarshalJSON(t *testing.T) {
	Convey("Given a list of valid responses", t, func() {
		tests := []struct {
			input  response
			output []byte
		}{
			{
				response{
					Error: &responseError{
						Title:  "error during query",
						Detail: "detail from error",
					},
				},
				infraTest.Load("response.1.json"),
			},
			{
				response{
					Resource: &resource{
						ID:        "123",
						Addresses: []string{"http://address1", "https://address2"},
						Path:      "/products/{*}",
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Change: flare.ResourceChange{
							Kind:  flare.ResourceChangeInteger,
							Field: "version",
						},
					},
				},
				infraTest.Load("response.2.json"),
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
							Change: flare.ResourceChange{
								Kind:  flare.ResourceChangeInteger,
								Field: "version",
							},
						},
					},
				},
				infraTest.Load("response.3.json"),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)

				c1, c2 := make(map[string]interface{}), make(map[string]interface{})
				err = json.Unmarshal([]byte(content), &c1)
				So(err, ShouldBeNil)

				err = json.Unmarshal([]byte(tt.output), &c2)
				So(err, ShouldBeNil)

				So(c1, ShouldResemble, c2)
			}
		})
	})
}

func TestResourceCreateValidAddresses(t *testing.T) {
	Convey("Given a list of valid addresses", t, func() {
		tests := []resourceCreate{
			{Addresses: []string{"http://app.io"}},
			{Addresses: []string{"https://app.com"}},
			{Addresses: []string{"http://app.io", "https://app.com"}},
		}

		Convey("The validation should not return a error", func() {
			for _, tt := range tests {
				result := tt.validAddresses()
				So(result, ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid addresses", t, func() {
		tests := []resourceCreate{
			{},
			{Addresses: []string{""}},
			{Addresses: []string{"tcp://127.0.0.1:8080"}},
			{Addresses: []string{"%zzzzz"}},
			{Addresses: []string{"http://app,com/teste"}},
			{Addresses: []string{"http://app,com#fragment"}},
			{Addresses: []string{"http://app,com?project=flare"}},
		}

		Convey("The validation should return a error", func() {
			for _, tt := range tests {
				result := tt.validAddresses()
				So(result, ShouldNotBeNil)
			}
		})
	})
}

func TestResourceCreateValidWildcard(t *testing.T) {
	Convey("Given a list of valid wildcards", t, func() {
		tests := []resourceCreate{
			{Path: "/users/{*}"},
			{Path: "/users/{*}/{wildcard2}"},
			{Path: "/{wildcard1}/{wildcard2}/{wildcard3}"},
			{Path: "/{wildcard}/{*}/users"},
		}

		Convey("The validation should not return a error", func() {
			for _, tt := range tests {
				result := tt.validWildcard()
				So(result, ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid wildcards", t, func() {
		tests := []resourceCreate{
			{Path: "/users"},
			{Path: "/{*}{*}"},
			{Path: "/{wildcard}}"},
			{Path: "/{*}/{*}"},
			{Path: "/{wildcard}/{*}/{wildcard}"},
		}

		Convey("The validation should return a error", func() {
			for _, tt := range tests {
				result := tt.validWildcard()
				So(result, ShouldNotBeNil)
			}
		})
	})
}

func TestResourceCreateValid(t *testing.T) {
	Convey("Given a list of valid resourceCreate", t, func() {
		tests := []resourceCreate{
			{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change: resourceCreateChange{
					Field:      "updatedAt",
					Kind:       flare.ResourceChangeDate,
					DateFormat: "2006-01-02T15:04:05Z07:00",
				},
			},
			{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "incrCounter", Kind: flare.ResourceChangeInteger},
			},
		}

		Convey("The validation should not return a error", func() {
			for _, tt := range tests {
				result := tt.valid()
				So(result, ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid resourceCreate", t, func() {
		tests := []resourceCreate{
			{},
			{Addresses: []string{"http://app.com"}},
			{Addresses: []string{"http://app.com"}, Path: "users"},
			{Addresses: []string{"http://app.com"}, Path: "/users"},
			{Addresses: []string{"http://app.com"}, Path: "/users/{*}-path/posts/{*}"},
			{Addresses: []string{"http://app.com"}, Path: "/users/{*}"},
			{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "updatedAt"},
			},
			{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "updatedAt", Kind: flare.ResourceChangeDate},
			},
			{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}/",
				Change:    resourceCreateChange{Field: "incrCounter", Kind: flare.ResourceChangeInteger},
			},
		}

		Convey("The validation should return a error", func() {
			for _, tt := range tests {
				result := tt.valid()
				So(result, ShouldNotBeNil)
			}
		})
	})
}
