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
		input  flare.Pagination
		output string
		hasErr bool
	}{
		{
			"Should pass",
			flare.Pagination{Limit: 30, Offset: 0},
			`{"limit":30,"offset":0,"total":0}`,
			false,
		},
		{
			"Should pass",
			flare.Pagination{Limit: 10, Offset: 30, Total: 120},
			`{"limit":10,"offset":30,"total":120}`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pagination{&tt.input}
			content, err := p.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("pagination.MarshalJSON error result, want '%v', got '%v'", tt.hasErr, (err != nil))
				t.FailNow()
			}

			if string(content) != tt.output {
				t.Errorf("pagination.MarshalJSON, want '%v', got '%v'", string(content), tt.output)
			}
		})
	}
}

func TestResourceMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  flare.Resource
		output string
		hasErr bool
	}{
		{
			"Should pass",
			flare.Resource{
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
			"Should pass",
			flare.Resource{
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
			r := resource{&tt.input}
			content, err := r.MarshalJSON()
			if tt.hasErr != (err != nil) {
				t.Errorf("resource.MarshalJSON error result, want '%v', got '%v'", tt.hasErr, (err != nil))
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
				t.Errorf("resource.MarshalJSON, want '%v', got '%v'", c2, c1)
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
			"Should pass",
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
			"Should pass",
			response{
				Resource: &resource{base: &flare.Resource{
					Id:        "123",
					Addresses: []string{"http://address1", "https://address2"},
					Path:      "/products/{track}",
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					Change: flare.ResourceChange{
						Kind:  flare.ResourceChangeInteger,
						Field: "version",
					},
				}},
			},
			`{"id":"123","addresses":["http://address1","https://address2"],"path":"/products/{track}",
			"change":{"field":"version","kind":"integer"},"createdAt":"2009-11-10T23:00:00Z"}`,
			false,
		},
		{
			"Should pass",
			response{
				Pagination: &pagination{base: &flare.Pagination{Limit: 10, Total: 30, Offset: 20}},
				Resources: []resource{
					{
						base: &flare.Resource{
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
				t.Errorf("response.MarshalJSON error result, want '%v', got '%v'", tt.hasErr, (err != nil))
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
				t.Errorf("response.MarshalJSON, want '%v', got '%v'", c2, c1)
			}
		})
	}
}
