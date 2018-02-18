// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
)

func TestDocumentMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal the document to JSON", t, func() {
		Convey("Given a valid document", func() {
			Convey("Should output a valid JSON", func() {
				d := document{
					ID:        url.URL{Scheme: "http", Host: "flare", Path: "/users/123"},
					Revision:  1,
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Content:   map[string]interface{}{},
				}
				expected := `{"id":"http://flare/users/123","updatedAt":"2009-11-10T23:00:00Z","content":{}}`

				content, err := d.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(content), ShouldEqual, expected)
			})
		})
	})
}

func TestDocumentParseBody(t *testing.T) {
	Convey("Feature: Parse the document body", t, func() {
		Convey("Given a list of valid bodies", func() {
			tests := []struct {
				content  string
				expected map[string]interface{}
			}{
				{
					`{"a": "b", "c": 1}`,
					map[string]interface{}{"a": "b", "c": float64(1)},
				},
				{
					`{"a": ["1", "2", "3"], "b": 1.2}`,
					map[string]interface{}{"a": []interface{}{"1", "2", "3"}, "b": 1.2},
				},
			}

			Convey("Expected to not have a error", func() {
				for _, tt := range tests {
					var d document
					err := d.parseBody(bytes.NewBufferString(tt.content))
					So(err, ShouldBeNil)
					So(d.Content, ShouldResemble, tt.expected)
				}
			})
		})

		Convey("Given a list of invalid bodies", func() {
			tests := []string{
				`["1", "2", 3]`,
				"",
			}

			Convey("Expected to have a error", func() {
				for _, tt := range tests {
					var d document
					err := d.parseBody(bytes.NewBufferString(tt))
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestDocumentParseRevision(t *testing.T) {
	Convey("Feature: Parse the document revision", t, func() {
		Convey("Given a list of valid documents", func() {
			tests := []struct {
				document document
				expected int
			}{
				{
					document{
						Content:  map[string]interface{}{"revision": float64(1)},
						Resource: flare.Resource{Change: flare.ResourceChange{Field: "revision"}},
					},
					1,
				},
				{
					document{
						Content: map[string]interface{}{"updatedAt": "2010-12-10T08:35:08.008Z"},
						Resource: flare.Resource{Change: flare.ResourceChange{
							Field:  "updatedAt",
							Format: "2006-01-02T15:04:05Z07:00",
						}},
					},
					1291970108008000000,
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(tt.document.parseRevision(), ShouldBeNil)
					So(tt.document.Revision, ShouldEqual, tt.expected)
				}
			})
		})

		Convey("Given a list of invalid documents", func() {
			tests := []struct {
				document document
				expected int
			}{
				{
					document{
						Content:  map[string]interface{}{"revision": 1},
						Resource: flare.Resource{Change: flare.ResourceChange{Field: "revision"}},
					},
					0,
				},
				{
					document{
						Content:  map[string]interface{}{"revision": "something"},
						Resource: flare.Resource{Change: flare.ResourceChange{Field: "revision"}},
					},
					0,
				},
				{
					document{
						Content: map[string]interface{}{"updatedAt": "2010-12-10T08:35:08.008Z"},
						Resource: flare.Resource{Change: flare.ResourceChange{
							Field:  "updatedAt",
							Format: "invalid",
						}},
					},
					0,
				},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(tt.document.parseRevision(), ShouldNotBeNil)
				}
			})
		})
	})
}

func TestValidEndpoint(t *testing.T) {
	Convey("Feature: Validate if the document id is valid", t, func() {
		Convey("Given a list of valid endpoints", func() {
			tests := []url.URL{
				{
					Scheme: "http", Host: "flare", Path: "/users/123",
				},
			}

			Convey("Should not generate a error", func() {
				for _, tt := range tests {
					So(validEndpoint(&tt), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid endpoints", func() {
			tests := []url.URL{
				{
					Scheme: "http", Host: "flare", Path: "/users/123", Opaque: "opaque",
				},
				{
					Scheme: "http", Host: "flare", Path: "/users/123", User: &url.Userinfo{},
				},
				{
					Scheme: "http", Path: "/users/123",
				},
				{
					Scheme: "http", Host: "flare",
				},
				{
					Scheme: "http", Host: "flare", Path: "/users/123", RawQuery: "filter=true",
				},
				{
					Scheme: "http", Host: "flare", Path: "/users/123", Fragment: "#something",
				},
				{
					Scheme: "ftp", Host: "flare", Path: "/users/123",
				},
				{
					Host: "flare", Path: "/users/123",
				},
			}

			Convey("Should not generate a error", func() {
				for _, tt := range tests {
					So(validEndpoint(&tt), ShouldNotBeNil)
				}
			})
		})
	})
}
