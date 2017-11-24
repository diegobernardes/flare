// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
)

func TestDocumentMarshalJSON(t *testing.T) {
	Convey("Given a list of valid documents", t, func() {
		tests := []struct {
			input  document
			output string
		}{
			{
				document{
					ID:        "123",
					Revision:  1,
					UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Content:   map[string]interface{}{},
				},
				`{"id":"123","updatedAt":"2009-11-10T23:00:00Z","content":{}}`,
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

func TestParseDocument(t *testing.T) {
	Convey("Given a list of valid documents", t, func() {
		tests := []struct {
			id       string
			content  []byte
			resource flare.Resource
			output   flare.Document
		}{
			{
				"http://app.com/users/123",
				[]byte(`{"seq":1}`),
				flare.Resource{Change: flare.ResourceChange{Field: "seq"}},
				flare.Document{
					ID:       "http://app.com/users/123",
					Revision: 1,
					Content:  map[string]interface{}{"seq": float64(1)},
				},
			},
			{
				"http://app.com/users/123",
				[]byte(`{"seq":"2006-01-02"}`),
				flare.Resource{Change: flare.ResourceChange{Field: "seq", Format: "2006-01-02"}},
				flare.Document{
					ID:       "http://app.com/users/123",
					Revision: 1136160000000000000,
					Content:  map[string]interface{}{"seq": "2006-01-02"},
				},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				document, err := parseDocument(tt.id, tt.content, &tt.resource)
				So(err, ShouldBeNil)
				tt.output.UpdatedAt = document.UpdatedAt
				tt.output.Resource = tt.resource
				So(*document, ShouldResemble, tt.output)
			}
		})
	})

	Convey("Given a list of invalid documents", t, func() {
		tests := []struct {
			id       string
			content  []byte
			resource flare.Resource
			output   flare.Document
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

		Convey("The output should generate a error", func() {
			for _, tt := range tests {
				_, err := parseDocument(tt.id, tt.content, &tt.resource)
				So(err, ShouldNotBeNil)
			}
		})
	})
}
