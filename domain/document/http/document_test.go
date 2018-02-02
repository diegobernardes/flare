// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
)

func TestDocumentMarshalJSON(t *testing.T) {
	Convey("Feature: Marshal the document to JSON", t, func() {
		Convey("Should output a valid JSON", func() {
			d := document{
				ID:        "123",
				Revision:  1,
				UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Content:   map[string]interface{}{},
			}
			expected := `{"id":"123","updatedAt":"2009-11-10T23:00:00Z","content":{}}`

			content, err := d.MarshalJSON()
			So(err, ShouldBeNil)
			So(string(content), ShouldEqual, expected)

		})
	})
}

func TestParseDocument(t *testing.T) {
	Convey("Feature: Parse a []byte to a document", t, func() {
		Convey("Given a list of valid documents", func() {
			tests := []struct {
				id       string
				content  []byte
				resource flare.Resource
				expected flare.Document
			}{
				{
					"http://app.com/users/123",
					[]byte(`{"sequence":1}`),
					flare.Resource{Change: flare.ResourceChange{Field: "sequence"}},
					flare.Document{
						ID:       "http://app.com/users/123",
						Revision: 1,
						Content:  map[string]interface{}{"sequence": float64(1)},
					},
				},
				{
					"http://app.com/users/123",
					[]byte(`{"sequence":"2006-01-02"}`),
					flare.Resource{Change: flare.ResourceChange{Field: "sequence", Format: "2006-01-02"}},
					flare.Document{
						ID:       "http://app.com/users/123",
						Revision: 1136160000000000000,
						Content:  map[string]interface{}{"sequence": "2006-01-02"},
					},
				},
			}

			Convey("Should output a valid document", func() {
				for _, tt := range tests {
					document, err := parseDocument(bytes.NewBuffer(tt.content), tt.id, &tt.resource)
					So(err, ShouldBeNil)
					So(document, ShouldNotEqual, time.Time{})

					tt.expected.UpdatedAt = document.UpdatedAt
					tt.expected.Resource = tt.resource
					So(*document, ShouldResemble, tt.expected)
				}
			})
		})

		Convey("Given a list of invalid documents", func() {
			tests := []struct {
				id       string
				content  []byte
				resource flare.Resource
				expected flare.Document
			}{
				{
					"http://app.com/users/123",
					[]byte(``),
					flare.Resource{},
					flare.Document{},
				},
				{
					"http://app.com/users/123",
					[]byte(`{"seq":{}}`),
					flare.Resource{Change: flare.ResourceChange{Field: "seq"}},
					flare.Document{},
				},
				{
					"http://app.com/users/123",
					[]byte(`{"seq":"2006-01-02"}`),
					flare.Resource{Change: flare.ResourceChange{Field: "seq"}},
					flare.Document{},
				},
			}

			Convey("Should output a error", func() {
				for _, tt := range tests {
					_, err := parseDocument(bytes.NewBuffer(tt.content), tt.id, &tt.resource)
					So(err, ShouldNotBeNil)
				}
			})
		})
	})
}
