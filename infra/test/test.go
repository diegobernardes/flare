// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
)

// Load is used by tests to load mocks. It used the runtime.Caller to get the request file directory
// and load all the files from a testdata folder at the same level.
func Load(name string) []byte {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get the caller that invoked load")
	}

	path := fmt.Sprintf("%s/testdata/%s", filepath.Dir(file), name)
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

// CompareJSONBytes take two arrays of byte, marshal then to a json struct and then
// compare if they are equal.
func CompareJSONBytes(a, b []byte) {
	c1, c2 := make(map[string]interface{}), make(map[string]interface{})
	err := json.Unmarshal(a, &c1)
	convey.So(err, convey.ShouldBeNil)

	err = json.Unmarshal(b, &c2)
	convey.So(err, convey.ShouldBeNil)

	convey.So(c1, convey.ShouldResemble, c2)
}
