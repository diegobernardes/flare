// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDocumentNewer(t *testing.T) {
	Convey("Feature: Check if a document is newer", t, func() {
		Convey("Given a list of newer documents", func() {
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

			Convey("Should be newer", func() {
				for _, tt := range tests {
					So(tt.target.Newer(tt.reference), ShouldBeTrue)
				}
			})
		})

		Convey("Given a list of older documents", func() {
			tests := []struct {
				reference *Document
				target    Document
			}{
				{
					&Document{Revision: 2},
					Document{Revision: 1},
				},
			}

			Convey("Should be older", func() {
				for _, tt := range tests {
					So(tt.target.Newer(tt.reference), ShouldBeFalse)
				}
			})
		})
	})
}
