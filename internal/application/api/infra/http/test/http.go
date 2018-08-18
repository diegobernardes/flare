package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/smartystreets/goconvey/convey" // nolint
)

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
	So(err, ShouldBeNil)
	So(resp.StatusCode, ShouldEqual, status)
	So(resp.Header, ShouldResemble, header)

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
