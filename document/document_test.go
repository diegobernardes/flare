// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDocumentMarshalJSON(t *testing.T) {
	Convey("Given a list of valid documents", t, func() {
		tests := []struct {
			input  document
			output string
		}{
			{
				document{
					Id:               "123",
					ChangeFieldValue: "1",
					UpdatedAt:        time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				},
				`{"id":"123","changeFieldValue":"1","updatedAt":"2009-11-10T23:00:00Z"}`,
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
