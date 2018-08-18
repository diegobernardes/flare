package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/smartystreets/goconvey/convey" // nolint
)

// Load is used by tests to load mocks. It used the runtime.Caller to get the request file directory
// and load all the files from a testdata folder at the same level.
func Load(name string) []byte {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		So(errors.New("could not get the caller that invoked load"), ShouldBeNil)
	}

	path := fmt.Sprintf("%s/testdata/%s", filepath.Dir(file), name)
	f, err := os.Open(path)
	So(err, ShouldBeNil)

	content, err := ioutil.ReadAll(f)
	So(err, ShouldBeNil)
	return content
}

// CompareJSONBytes take two arrays of byte, marshal then to a json struct and then
// compare if they are equal.
func CompareJSONBytes(a, b []byte) {
	c1, c2 := make(map[string]interface{}), make(map[string]interface{})
	err := json.Unmarshal(a, &c1)
	So(err, ShouldBeNil)

	err = json.Unmarshal(b, &c2)
	So(err, ShouldBeNil)

	So(c1, ShouldResemble, c2)
}
