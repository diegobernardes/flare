// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/diegobernardes/flare"
)

func TestPaginationMarshalJSON(t *testing.T) {
	Convey("Given a list or valid paginations", t, func() {
		tests := []struct {
			input  pagination
			output string
		}{
			{
				pagination{Limit: 30, Offset: 0},
				`{"limit":30,"offset":0,"total":0}`,
			},
			{
				pagination{Limit: 10, Offset: 30, Total: 120},
				`{"limit":10,"offset":30,"total":120}`,
			},
		}

		Convey("Output should be valid", func() {
			for _, tt := range tests {
				content, err := tt.input.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(content), ShouldEqual, tt.output)
			}
		})
	})
}

func TestResourceMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  resource
		output string
		hasErr bool
	}{
		{
			"Valid JSON",
			resource{
				Id:        "id",
				CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Addresses: []string{"http://flare.io", "https://flare.com"},
				Path:      "/resources/{*}",
				Change: flare.ResourceChange{
					Field: "version",
					Kind:  flare.ResourceChangeInteger,
				},
			},
			`{"id":"id","addresses":["http://flare.io","https://flare.com"],"path":"/resources/{*}",
			"change":{"field":"version","kind":"integer"},"createdAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
		{
			"Valid JSON",
			resource{
				Id:        "id",
				CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Addresses: []string{"http://flare.io", "https://flare.com"},
				Path:      "/resources/{*}",
				Change: flare.ResourceChange{
					Field:      "updatedAt",
					Kind:       flare.ResourceChangeDate,
					DateFormat: "2006-01-02",
				},
			},
			`{"id":"id","addresses":["http://flare.io","https://flare.com"],"path":"/resources/{*}",
			"change":{"field":"updatedAt","kind":"date","dateFormat":"2006-01-02"},
			"createdAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("resource.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			c1, c2 := make(map[string]interface{}), make(map[string]interface{})
			if err := json.Unmarshal([]byte(content), &c1); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c1, content,
				)))
				t.FailNow()
			}

			if err := json.Unmarshal([]byte(tt.output), &c2); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c2, tt.output,
				)))
				t.FailNow()
			}

			if !reflect.DeepEqual(c1, c2) {
				t.Errorf("resource.MarshalJSON invalid result, want '%v', got '%v'", c2, c1)
			}
		})
	}
}

func TestResponseMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  response
		output string
		hasErr bool
	}{
		{
			"Valid JSON",
			response{
				Error: &responseError{
					Status: http.StatusBadRequest,
					Title:  "error during query",
					Detail: "detail from error",
				},
			},
			`{"error":{"status":400,"title":"error during query","detail":"detail from error"}}`,
			false,
		},
		{
			"Valid JSON",
			response{
				Resource: &resource{
					Id:        "123",
					Addresses: []string{"http://address1", "https://address2"},
					Path:      "/products/{*}",
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Change: flare.ResourceChange{
						Kind:  flare.ResourceChangeInteger,
						Field: "version",
					},
				},
			},
			`{"id":"123","addresses":["http://address1","https://address2"],"path":"/products/{*}",
			"change":{"field":"version","kind":"integer"},"createdAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
		{
			"Valid JSON",
			response{
				Pagination: (*pagination)(&flare.Pagination{Limit: 10, Total: 30, Offset: 20}),
				Resources: []resource{
					{
						Id:        "123",
						Addresses: []string{"http://address1", "https://address2"},
						Path:      "/products/{*}",
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Change: flare.ResourceChange{
							Kind:  flare.ResourceChangeInteger,
							Field: "version",
						},
					},
				},
			},
			`{"resources":[{"id":"123","addresses":["http://address1","https://address2"],
			"path":"/products/{*}","change":{"field":"version","kind":"integer"},
			"createdAt":"2009-11-10T23:00:00Z"}],"pagination":{"limit":10,"offset":20,"total":30}}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("response.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			c1, c2 := make(map[string]interface{}), make(map[string]interface{})
			if err := json.Unmarshal([]byte(content), &c1); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c1, content,
				)))
				t.FailNow()
			}

			if err := json.Unmarshal([]byte(tt.output), &c2); err != nil {
				t.Error(errors.Wrap(err, fmt.Sprintf(
					"error during json.Unmarshal to '%v' with value '%s'", c2, tt.output,
				)))
				t.FailNow()
			}

			if !reflect.DeepEqual(c1, c2) {
				t.Errorf("response.MarshalJSON invalid result, want '%v', got '%v'", c2, c1)
			}
		})
	}
}

func TestResourceCreateValidAddresses(t *testing.T) {
	tests := []struct {
		name   string
		input  resourceCreate
		hasErr bool
	}{
		{
			"Empty addresses",
			resourceCreate{},
			true,
		},
		{
			"Valid addresses",
			resourceCreate{Addresses: []string{"http://app.io", "https://app.com"}},
			false,
		},
		{
			"Missing schema",
			resourceCreate{Addresses: []string{""}},
			true,
		},
		{
			"Invalid schema",
			resourceCreate{Addresses: []string{"tcp://127.0.0.1:8080"}},
			true,
		},
		{
			"Invalid address",
			resourceCreate{Addresses: []string{"%zzzzz"}},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app,com/teste"}},
			true,
		},
		{
			"Invalid fragment",
			resourceCreate{Addresses: []string{"http://app,com#fragment"}},
			true,
		},
		{
			"Invalid query string",
			resourceCreate{Addresses: []string{"http://app,com?project=flare"}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.validAddresses()
			if tt.hasErr != (result != nil) {
				t.Errorf("resourceCreate.validAddresses invalid result, want '%v', got '%v'", tt.hasErr, result)
			}
		})
	}
}

func TestResourceCreateValidWildcard(t *testing.T) {
	tests := []struct {
		name   string
		input  resourceCreate
		hasErr bool
	}{
		{
			"Valid wildcard",
			resourceCreate{Path: "/users/{*}"},
			false,
		},
		{
			"Invalid wildcard",
			resourceCreate{Path: "/users{*}"},
			true,
		},
		{
			"Invalid wildcard",
			resourceCreate{Path: "/{*}{*}"},
			true,
		},
		{
			"Invalid wildcard",
			resourceCreate{Path: "/{wildcard}}"},
			true,
		},
		{
			"Invalid wildcard",
			resourceCreate{Path: "/{*}/{*}"},
			true,
		},
		{
			"Invalid wildcard",
			resourceCreate{Path: "/{wildcard}/{*}/{wildcard}"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.validWildcard()
			if tt.hasErr != (result != nil) {
				t.Errorf("resourceCreate.validWildcard invalid result, want '%v', got '%v'", tt.hasErr, result)
			}
		})
	}
}

func TestResourceCreateValid(t *testing.T) {
	tests := []struct {
		name   string
		input  resourceCreate
		hasErr bool
	}{
		{
			"Invalid addresses",
			resourceCreate{},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app.com"}},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "users"},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users"},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users/{*}-path/posts/{*}"},
			true,
		},
		{
			"Invalid change",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users/{*}"},
			true,
		},
		{
			"Invalid change kind",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "updatedAt"},
			},
			true,
		},
		{
			"Missing date format",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "updatedAt", Kind: flare.ResourceChangeDate},
			},
			true,
		},
		{
			"Valid resource",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change:    resourceCreateChange{Field: "incrCounter", Kind: flare.ResourceChangeInteger},
			},
			false,
		},
		{
			"Valid resource",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{*}",
				Change: resourceCreateChange{
					Field:      "updatedAt",
					Kind:       flare.ResourceChangeDate,
					DateFormat: "2006-01-02T15:04:05Z07:00",
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.valid()
			if tt.hasErr != (result != nil) {
				t.Errorf("resourceCreate.valid invalid result, want '%v', got '%v'", tt.hasErr, result)
			}
		})
	}
}
