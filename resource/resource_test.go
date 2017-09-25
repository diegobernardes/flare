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

	"github.com/diegobernardes/flare"
)

func TestPaginationMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  pagination
		output string
		hasErr bool
	}{
		{
			"Valid pagination",
			pagination{Limit: 30, Offset: 0},
			`{"limit":30,"offset":0,"total":0}`,
			false,
		},
		{
			"Valid pagination",
			pagination{Limit: 10, Offset: 30, Total: 120},
			`{"limit":10,"offset":30,"total":120}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.input.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("pagination.MarshalJSON invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			if string(content) != tt.output {
				t.Errorf(
					"pagination.MarshalJSON invalid result, want '%v', got '%v'", string(content), tt.output,
				)
			}
		})
	}
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
				Path:      "/resources/{track}",
				Change: flare.ResourceChange{
					Field: "version",
					Kind:  flare.ResourceChangeInteger,
				},
			},
			`{"id":"id","addresses":["http://flare.io","https://flare.com"],"path":"/resources/{track}",
			"change":{"field":"version","kind":"integer"},"createdAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
		{
			"Valid JSON",
			resource{
				Id:        "id",
				CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Addresses: []string{"http://flare.io", "https://flare.com"},
				Path:      "/resources/{track}",
				Change: flare.ResourceChange{
					Field:      "updatedAt",
					Kind:       flare.ResourceChangeDate,
					DateFormat: "2006-01-02",
				},
			},
			`{"id":"id","addresses":["http://flare.io","https://flare.com"],"path":"/resources/{track}",
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
					Path:      "/products/{track}",
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Change: flare.ResourceChange{
						Kind:  flare.ResourceChangeInteger,
						Field: "version",
					},
				},
			},
			`{"id":"123","addresses":["http://address1","https://address2"],"path":"/products/{track}",
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
						Path:      "/products/{track}",
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Change: flare.ResourceChange{
							Kind:  flare.ResourceChangeInteger,
							Field: "version",
						},
					},
				},
			},
			`{"resources":[{"id":"123","addresses":["http://address1","https://address2"],
			"path":"/products/{track}","change":{"field":"version","kind":"integer"},
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

func TestResourceCreateValidTrack(t *testing.T) {
	tests := []struct {
		name   string
		input  resourceCreate
		hasErr bool
	}{
		{
			"Missing track",
			resourceCreate{Path: "/users/{*}"},
			true,
		},
		{
			"Empty path",
			resourceCreate{Path: ""},
			true,
		},
		{
			"Missing track",
			resourceCreate{Path: "/users/{*}/posts/{*}"},
			true,
		},
		{
			"2 track tags",
			resourceCreate{Path: "/users/{track}/posts/{track}"},
			true,
		},
		{
			"Wildcard before track",
			resourceCreate{Path: "/users/{track}/posts/{*}"},
			true,
		},
		{
			"Valid track",
			resourceCreate{Path: "/users/{*}/posts/{track}"},
			false,
		},
		{
			"Valid track",
			resourceCreate{Path: "/users/{track}"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.validTrack()
			if tt.hasErr != (result != nil) {
				t.Errorf("resourceCreate.validTrack invalid result, want '%v', got '%v'", tt.hasErr, result)
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
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users"},
			true,
		},
		{
			"Invalid path",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users/{*}-path/posts/{track}"},
			true,
		},
		{
			"Invalid change",
			resourceCreate{Addresses: []string{"http://app.com"}, Path: "/users/{track}"},
			true,
		},
		{
			"Invalid change kind",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{track}",
				Change:    resourceCreateChange{Field: "updatedAt"},
			},
			true,
		},
		{
			"Missing date format",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{track}",
				Change:    resourceCreateChange{Field: "updatedAt", Kind: flare.ResourceChangeDate},
			},
			true,
		},
		{
			"Valid resource",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{track}",
				Change:    resourceCreateChange{Field: "incrCounter", Kind: flare.ResourceChangeInteger},
			},
			false,
		},
		{
			"Valid resource",
			resourceCreate{
				Addresses: []string{"http://app.com"},
				Path:      "/users/{track}",
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
