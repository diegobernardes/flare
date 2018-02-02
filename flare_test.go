// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPaginationValid(t *testing.T) {
	Convey("Feature: Validate the Pagination", t, func() {
		Convey("Given a list of valid paginations", func() {
			tests := []Pagination{
				{},
				{Limit: 1},
				{Offset: 1},
				{Limit: 30, Offset: 10},
			}

			Convey("Should not return a error", func() {
				for _, tt := range tests {
					So(tt.Valid(), ShouldBeNil)
				}
			})
		})

		Convey("Given a list of invalid paginations", func() {
			tests := []struct {
				title      string
				pagination Pagination
			}{
				{
					"Should have a invalid offset (1)",
					Pagination{Offset: -1},
				},
				{
					"Should have a invalid offset (2)",
					Pagination{Limit: 1, Offset: -1},
				},
				{
					"Should have a invalid limit (1)",
					Pagination{Limit: -1},
				},
				{
					"Should have a invalid limit (2)",
					Pagination{Offset: 1, Limit: -1},
				},
			}

			for _, tt := range tests {
				Convey(tt.title, func() {
					So(tt.pagination.Valid(), ShouldNotBeNil)
				})
			}
		})
	})
}
