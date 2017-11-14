// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWorkerMarshal(t *testing.T) {
	Convey("Given a list of valid params", t, func() {
		tests := []struct {
			id     string
			action string
			body   []byte
			output []byte
		}{
			{
				"123",
				"update",
				[]byte("{}"),
				[]byte(`{"id":"123", "action":"update", "body":"{}"}`),
			},
			{
				"123",
				"update",
				[]byte(`{"content":"data"}`),
				[]byte(`{"id":"123", "action":"update", "body":"{\"content\":\"data\"}"}`),
			},
		}

		Convey("The output should be valid", func() {
			for _, tt := range tests {
				w := &Worker{}
				content, err := w.marshal(tt.id, tt.action, tt.body)
				So(err, ShouldBeNil)

				b1, b2 := make(map[string]interface{}), make(map[string]interface{})
				err = json.Unmarshal(content, &b1)
				So(err, ShouldBeNil)

				err = json.Unmarshal(tt.output, &b2)
				So(err, ShouldBeNil)

				So(b1, ShouldResemble, b2)
			}
		})
	})
}
