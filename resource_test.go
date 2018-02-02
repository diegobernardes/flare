// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResourceChangeValid(t *testing.T) {
	Convey("Feature: Validate the ResourceChange", t, func() {
		Convey("Given a list of valid resource changes", func() {
			tests := []ResourceChange{
				{Field: "updatedAt", Format: "2006-01-02"},
				{Field: "sequence"},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(tt.Valid(), ShouldBeNil)
				}
			})
		})

		Convey("Given a invalid resource change", func() {
			Convey("Should return a error", func() {
				r := ResourceChange{}
				So(r.Valid(), ShouldNotBeNil)
			})
		})
	})
}
