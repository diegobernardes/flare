// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDocumentValid(t *testing.T) {
	Convey("Given a list of valid documents", t, func() {
		tests := []Document{
			{
				ID:       "1",
				Resource: Resource{Change: ResourceChange{Field: "seq", Kind: ResourceChangeInteger}},
			},
		}

		Convey("The validation should not return a error", func() {
			for _, tt := range tests {
				So(tt.Valid(), ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid documents", t, func() {
		tests := []struct {
			title string
			doc   Document
		}{
			{
				"Should have a invalid id 1",
				Document{},
			},
			{
				"Should have a invalid id 2",
				Document{
					Resource: Resource{Change: ResourceChange{Field: "seq", Kind: ResourceChangeInteger}},
				},
			},
			{
				"Should have a invalid change",
				Document{
					ID:       "1",
					Resource: Resource{Change: ResourceChange{Field: "updatedAt", Kind: ResourceChangeDate}},
				},
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				So(tt.doc.Valid(), ShouldNotBeNil)
			})
		}
	})
}

func TestDocumentNewer(t *testing.T) {
	Convey("Given a list of newer documents", t, func() {
		tests := []struct {
			reference *Document
			target    Document
		}{
			{
				nil,
				Document{},
			},
			{
				&Document{Revision: 1},
				Document{Revision: 2},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				So(tt.target.Newer(tt.reference), ShouldBeTrue)
			}
		})
	})

	Convey("Given a list of older documents", t, func() {
		tests := []struct {
			reference *Document
			target    Document
		}{
			{
				&Document{Revision: 2},
				Document{Revision: 1},
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				So(tt.target.Newer(tt.reference), ShouldBeFalse)
			}
		})
	})
}
