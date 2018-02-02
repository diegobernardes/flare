// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
)

// Handler is used to test http handlers.
func Handler(
	status int,
	header http.Header,
	handler func(w http.ResponseWriter, r *http.Request),
	req *http.Request,
	expectedBody []byte,
) func(*testing.T) {
	return func(t *testing.T) {
		w := httptest.NewRecorder()
		handler(w, req)

		resp := w.Result()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf(errors.Wrap(err, "unexpected error").Error())
			t.FailNow()
		}

		if status != resp.StatusCode {
			t.Errorf("status invalid result, want '%v', got '%v'", status, resp.Status)
		}

		if !reflect.DeepEqual(header, resp.Header) {
			t.Errorf(
				"header invalid result, want '%v', got '%v'",
				pretty.Sprint(header), pretty.Sprint(resp.Header),
			)
		}

		if len(body) == 0 && expectedBody == nil {
			return
		}

		b1, b2 := make(map[string]interface{}), make(map[string]interface{})
		if err := json.Unmarshal(body, &b1); err != nil {
			t.Errorf(errors.Wrap(err, "unexpected error").Error())
			t.FailNow()
		}

		if err := json.Unmarshal(expectedBody, &b2); err != nil {
			t.Errorf(errors.Wrap(err, "unexpected error").Error())
			t.FailNow()
		}

		if !reflect.DeepEqual(b1, b2) {
			t.Errorf("body invalid result, want '%v', got '%v'", pretty.Sprint(b2), pretty.Sprint(b1))
		}
	}
}

// Runner is used to execute http requests and check if it matchs.
func Runner(
	status int,
	header http.Header,
	handler func(w http.ResponseWriter, r *http.Request),
	req *http.Request,
	expectedBody []byte,
) {
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, status)
	convey.So(resp.Header, convey.ShouldResemble, header)

	if len(body) == 0 && expectedBody == nil {
		return
	}

	b1, b2 := make(map[string]interface{}), make(map[string]interface{})
	err = json.Unmarshal(body, &b1)
	convey.So(err, convey.ShouldBeNil)

	err = json.Unmarshal(expectedBody, &b2)
	convey.So(err, convey.ShouldBeNil)

	convey.So(b1, convey.ShouldResemble, b2)
}
