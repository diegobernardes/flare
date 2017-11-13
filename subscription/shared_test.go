// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func load(name string) []byte {
	path := fmt.Sprintf("testdata/%s", name)
	f, err := os.Open(path)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during open '%s'", path)))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during read '%s'", path)))
	}
	return content
}

func httpRunner(
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
	So(err, ShouldBeNil)
	So(status, ShouldEqual, resp.StatusCode)
	So(header, ShouldResemble, resp.Header)

	if len(body) == 0 && expectedBody == nil {
		return
	}

	b1, b2 := make(map[string]interface{}), make(map[string]interface{})
	err = json.Unmarshal(body, &b1)
	So(err, ShouldBeNil)

	err = json.Unmarshal(expectedBody, &b2)
	So(err, ShouldBeNil)

	So(b1, ShouldResemble, b2)
}
