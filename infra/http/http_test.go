// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name         string
		defaultLimit int
		uri          url.URL
		pagination   *flare.Pagination
		hasErr       bool
	}{
		{
			"Invalid pagination",
			0,
			url.URL{RawQuery: "limit=sample"},
			nil,
			true,
		},
		{
			"Invalid pagination",
			0,
			url.URL{RawQuery: "offset=sample"},
			nil,
			true,
		},
		{
			"Invalid pagination",
			0,
			url.URL{RawQuery: "limit=1&offset=sample"},
			nil,
			true,
		},
		{
			"Invalid pagination",
			0,
			url.URL{RawQuery: "limit=sample&offset=1"},
			nil,
			true,
		},
		{
			"Valid pagination",
			0,
			url.URL{RawQuery: "limit=30&offset=60"},
			&flare.Pagination{Limit: 30, Offset: 60},
			false,
		},
		{
			"Valid pagination",
			10,
			url.URL{RawQuery: "offset=60"},
			&flare.Pagination{Limit: 10, Offset: 60},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pagination, err := ParsePagination(tt.defaultLimit)(&http.Request{URL: &tt.uri})
			if tt.hasErr != (err != nil) {
				t.Errorf("ParsePagination invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			if !reflect.DeepEqual(pagination, tt.pagination) {
				t.Errorf("ParsePagination invalid result, want '%v', got '%v'", tt.pagination, pagination)
			}
		})
	}
}

func TestWriteResponse(t *testing.T) {
	wr := WriteResponse(log.NewNopLogger())

	tests := []struct {
		name    string
		status  int
		header  http.Header
		content interface{}
	}{
		{
			"Success",
			http.StatusOK,
			map[string][]string{"Content-Type": {"application/json"}},
			map[string]interface{}{},
		},
		{
			"Success",
			http.StatusInternalServerError,
			map[string][]string{"Content-Type": {"application/json"}, "Cache-Control": {"no-cache"}},
			map[string]interface{}{"error": "error detail"},
		},
		{
			"Success",
			http.StatusNoContent,
			map[string][]string{"Content-Type": {"application/json"}, "Cache-Control": {"no-cache"}},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			wr(w, tt.content, tt.status, tt.header)

			if tt.status != w.Code {
				t.Errorf("status invalid result, want '%v', got '%v'", tt.status, w.Code)
				t.FailNow()
			}

			if !reflect.DeepEqual(tt.header, w.HeaderMap) {
				t.Errorf("header invalid result, want '%v', got '%v'", tt.header, w.HeaderMap)
				t.FailNow()
			}

			if tt.content == nil {
				return
			}

			content := make(map[string]interface{})
			if err := json.Unmarshal(w.Body.Bytes(), &content); err != nil {
				t.Error(errors.Wrap(err, "unexpected error").Error())
				t.FailNow()
			}

			if !reflect.DeepEqual(content, tt.content) {
				t.Errorf("body invalid result, want '%v', got '%v'", tt.content, content)
			}
		})
	}
}
