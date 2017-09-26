// Copyright 2017 Diego Bernardes. All rights reserved.
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

	"github.com/pkg/errors"
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
			t.Errorf("header invalid result, want '%v', got '%v'", header, resp.Header)
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
			t.Errorf("body invalid result, want '%v', got '%v'", b2, b1)
		}
	}
}
