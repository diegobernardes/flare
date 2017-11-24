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
			{Field: "updatedAt", Format: "2006-01-02"},
			{Field: "sequence"},
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
		}

		for _, tt := range tests {
			Convey(tt.title, func() {
				So(tt.rc.Valid(), ShouldNotBeNil)
			})
		}
	})
}
