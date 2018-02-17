// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wildcard

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValid(t *testing.T) {
	Convey("Feature: Validate wildcards", t, func() {
		Convey("Given a list of valid wildcards", func() {
			tests := []string{
				"/",
				"/users",
				"/users/{*}",
				"/users/{ * }",
				"/users/{*}/{wildcard2}",
				"/{wildcard1}/{wildcard2}/{wildcard3}",
				"/{wildcard}/{*}/users",
				"/{ id1 }/{ id2}",
				"/{*}-something",
				"/some{*}thing",
				"/{*}thing{id}",
				"/some{*}thing/{id}",
				"/some{x}/{id}",
				"/some{id}teste{id2}/{id3}",
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					result := Valid(tt)
					So(result, ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid wildcards", func() {
			tests := []string{
				"/users/{}",
				"/users/{",
				"/users/}",
				"/{revision}",
				"/{ revision }",
				"/{id1}{id2}",
				"/{{*}}",
				"/{wildcard}}",
			}

			Convey("Should return a error", func() {
				for _, tt := range tests {
					result := Valid(tt)
					So(result, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestValidURL(t *testing.T) {
	Convey("Feature: Validate endpoints with wildcards", t, func() {
		Convey("Given a list of invalid wildcards", func() {
			tests := []string{
				"/{ id }/{ id}",
				"/{*}{*}",
				"/{*}/{*}",
				"/{wildcard}/{*}/{wildcard}",
				"/content-{id}",
				"/content/{id}/user-{guid}",
			}

			Convey("Should return a error", func() {
				for _, tt := range tests {
					result := ValidURL(tt)
					So(result, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestReplace(t *testing.T) {
	Convey("Feature: Replace wildcard value", t, func() {
		Convey("Given a list of parameters", func() {
			tests := []struct {
				wildcard string
				content  map[string]string
				expected string
			}{
				{
					"sample",
					nil,
					"sample",
				},
				{
					"sample/{id}",
					map[string]string{"id": "1"},
					"sample/1",
				},
				{
					"sample/{id}/{id}",
					map[string]string{"id": "1"},
					"sample/1/1",
				},
				{
					"sample/{1}/{2}",
					map[string]string{"1": "1", "2": "2"},
					"sample/1/2",
				},

				{
					"sample/{ 1 }",
					map[string]string{"1": "1"},
					"sample/1",
				},
				{
					"sample/{ 1 }",
					map[string]string{"1": "1", "2": "2"},
					"sample/1",
				},
				{
					"sample/{1}",
					map[string]string{"1": "123"},
					"sample/123",
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					value := Replace(tt.wildcard, tt.content)
					So(value, ShouldEqual, tt.expected)
				}
			})
		})
	})
}

func TestPresent(t *testing.T) {
	Convey("Feature: Check if a content have a wildcard", t, func() {
		Convey("Given a list of parameters", func() {
			tests := []struct {
				wildcard string
				expected bool
			}{
				{
					"sample",
					false,
				},
				{
					"sample/{id}",
					true,
				},
				{
					"sample/{",
					true,
				},
				{
					"sample/}",
					true,
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(Present(tt.wildcard), ShouldEqual, tt.expected)
				}
			})
		})
	})
}

func TestExtract(t *testing.T) {
	Convey("Feature: Get a list of wildcards", t, func() {
		Convey("Given a list of parameters", func() {
			tests := []struct {
				wildcard string
				expected []string
			}{
				{
					"sample",
					nil,
				},
				{
					"sample/{id}",
					[]string{"id"},
				},
				{
					"sample/{id}{id2}",
					[]string{"id", "id2"},
				},
				{
					"sample/{id}/{id}",
					[]string{"id"},
				},
				{
					"sample/{id}/{id}/{id}",
					[]string{"id"},
				},
				{
					"sample/{1}/{2}",
					[]string{"1", "2"},
				},

				{
					"sample/{ 1 }",
					[]string{"1"},
				},
				{
					"sample/{ 1 }",
					[]string{"1"},
				},
				{
					"sample/{1}",
					[]string{"1"},
				},
				{
					"sample/{*}/flare/{id}/wildcard/{sample}/{id}",
					[]string{"*", "id", "sample"},
				},
			}

			Convey("Should return a list of wildcards", func() {
				for _, tt := range tests {
					value := Extract(tt.wildcard)
					So(value, ShouldResemble, tt.expected)
				}
			})
		})
	})
}

func TestExtractValue(t *testing.T) {
	Convey("Feature: Extract the wildcard value from a content", t, func() {
		Convey("Given a list of parameters", func() {
			tests := []struct {
				a        string
				b        string
				expected map[string]string
			}{
				{
					"sample",
					"sample",
					map[string]string{},
				},
				{
					"users/{id}",
					"users/123",
					map[string]string{"id": "123"},
				},
				{
					"users/{id}/{action}",
					"users/123/update",
					map[string]string{"id": "123", "action": "update"},
				},
				{
					"{id}/{action}",
					"123/update",
					map[string]string{"id": "123", "action": "update"},
				},
				{
					"service-{type}",
					"service-user",
					map[string]string{"type": "user"},
				},
				{
					"service-{type}/sample",
					"service-user/sample",
					map[string]string{"type": "user"},
				},
				{
					"{type} {domain}",
					"service user",
					map[string]string{"type": "service", "domain": "user"},
				},
				{
					"{type} {domain}",
					"service user domain",
					map[string]string{"type": "service", "domain": "user domain"},
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					value := ExtractValue(tt.a, tt.b)
					So(value, ShouldResemble, tt.expected)
				}
			})
		})
	})
}

func TestNormalize(t *testing.T) {
	Convey("Feature: Normalize a wildcard", t, func() {
		Convey("Given a list of parameters", func() {
			tests := []struct {
				wildcard string
				expected string
			}{
				{
					"sample",
					"sample",
				},
				{
					"users/{id}",
					"users/{id}",
				},
				{
					"users/{id}/{id2}",
					"users/{id}/{id2}",
				},
				{
					"users/{ id }/{  id2}",
					"users/{id}/{id2}",
				},
				{
					"{ id }-sample-{  id2}",
					"{id}-sample-{id2}",
				},
				{
					" { id } ",
					" {id} ",
				},
			}

			Convey("Should have the expected output", func() {
				for _, tt := range tests {
					So(Normalize(tt.wildcard), ShouldEqual, tt.expected)
				}
			})
		})
	})
}
