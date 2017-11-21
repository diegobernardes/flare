// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResourceChangeValid(t *testing.T) {
	Convey("Given a list of valid resource changes", t, func() {
		tests := []ResourceChange{
			{Field: "updatedAt", Kind: ResourceChangeDate, DateFormat: "2006-01-02"},
			{Field: "sequence", Kind: ResourceChangeInteger},
		}

		Convey("The validation should not return a error", func() {
			for _, tt := range tests {
				So(tt.Valid(), ShouldBeNil)
			}
		})
	})

	Convey("Given a list of invalid resource changes", t, func() {
		tests := []struct {
			title string
			rc    ResourceChange
		}{
			{
				"Should be missing the field",
				ResourceChange{},
			},
			{
				"Should be missing the kind",
				ResourceChange{Field: "updatedAt"},
			},
			{
				"Should be missing the format",
				ResourceChange{Field: "updatedAt", Kind: ResourceChangeDate},
			},
			{
				"Should not have a format when the kind is integer",
				ResourceChange{Field: "sequence", Kind: ResourceChangeInteger, DateFormat: "2006-01-02"},
			},
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				So(tt.rc.Valid(), ShouldNotBeNil)
			})
		}
	})
}
