// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
)

func load(name string) []byte {
	path := fmt.Sprintf("testdata/%s", name)
	f, err := os.Open(path)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during open testfile '%s'", path)))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error during read testfile '%s'", path)))
	}
	return content
}

func TestHandleIndex(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		status     int
		header     http.Header
		body       []byte
		repository resourceRepository
	}{
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources?limit=sample", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("invalidPaginationType1.json"),
			resourceRepository{},
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources?offset=sample", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("invalidPaginationType2.json"),
			resourceRepository{},
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources?limit=-1", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("invalidPaginationValue1.json"),
			resourceRepository{},
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources?offset=-1", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("invalidPaginationValue2.json"),
			resourceRepository{},
		},
		{
			"Error during query",
			httptest.NewRequest("GET", "http://resources", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("errorSearch.json"),
			resourceRepository{err: errors.New("error during repository search")},
		},
		{
			"Valid search",
			httptest.NewRequest("GET", "http://resources", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("validSearch1.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{
					{
						Id:        "1",
						Addresses: []string{"http://app1.com", "https://app1.io"},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Path:      "/resources/{track}",
						Change: flare.ResourceChange{
							Field:      "updatedAt",
							Kind:       flare.ResourceChangeDate,
							DateFormat: "2006-01-02T15:04:05Z07:00",
						},
					},
				},
			),
		},
		{
			"Valid search",
			httptest.NewRequest("GET", "http://resources?limit=10", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("validSearch2.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{
					{
						Id:        "1",
						Addresses: []string{"http://app1.com", "https://app1.io"},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Path:      "/resources/{track}",
						Change: flare.ResourceChange{
							Field:      "updatedAt",
							Kind:       flare.ResourceChangeDate,
							DateFormat: "2006-01-02T15:04:05Z07:00",
						},
					},
					{
						Id:        "2",
						Addresses: []string{"http://app2.com", "https://app2.io"},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Path:      "/resources/{track}",
						Change: flare.ResourceChange{
							Field:      "updatedAt",
							Kind:       flare.ResourceChangeDate,
							DateFormat: "2006-01-02T15:04:05Z07:00",
						},
					},
				},
			),
		},
		{
			"Valid search",
			httptest.NewRequest("GET", "http://resources?limit=10&offset=1", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("validSearch3.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{
					{
						Id:        "1",
						Addresses: []string{"http://app1.com", "https://app1.io"},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Path:      "/resources/{track}",
						Change: flare.ResourceChange{
							Field:      "updatedAt",
							Kind:       flare.ResourceChangeDate,
							DateFormat: "2006-01-02T15:04:05Z07:00",
						},
					},
					{
						Id:        "2",
						Addresses: []string{"http://app2.com", "https://app2.io"},
						CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
						Path:      "/resources/{track}",
						Change: flare.ResourceChange{
							Field:      "updatedAt",
							Kind:       flare.ResourceChangeDate,
							DateFormat: "2006-01-02T15:04:05Z07:00",
						},
					},
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(
				ServiceLogger(log.NewNopLogger()),
				ServiceRepository(&tt.repository),
				ServiceGetResourceId(func(*http.Request) string { return "" }),
				ServiceGetResourceURI(func(string) string { return "" }),
			)
			if err != nil {
				t.Error(errors.Wrap(err, "error during service initialization"))
			}

			w := httptest.NewRecorder()
			service.HandleIndex(w, tt.req)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf(errors.Wrap(err, "unexpected error").Error())
				t.FailNow()
			}

			if tt.status != resp.StatusCode {
				t.Errorf("status, want '%v', got '%v'", tt.status, resp.Status)
			}

			if !reflect.DeepEqual(tt.header, resp.Header) {
				t.Errorf("status, want '%v', got '%v'", tt.header, resp.Header)
			}

			b1, b2 := make(map[string]interface{}), make(map[string]interface{})
			if err := json.Unmarshal(tt.body, &b1); err != nil {
				t.Errorf(errors.Wrap(err, "unexpected error").Error())
				t.FailNow()
			}

			if err := json.Unmarshal(body, &b2); err != nil {
				t.Errorf(errors.Wrap(err, "unexpected error").Error())
				t.FailNow()
			}

			if !reflect.DeepEqual(b1, b2) {
				t.Errorf("body, want '%v', got '%v'", b1, b2)
			}
		})
	}
}

type resourceRepository struct {
	date time.Time
	base flare.ResourceRepositorier
	err  error
}

func (r *resourceRepository) FindAll(
	ctx context.Context, pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	if r.err != nil {
		return nil, nil, r.err
	}

	resources, page, err := r.base.FindAll(ctx, pagination)
	if err != nil {
		return nil, nil, err
	}

	for i := range resources {
		resources[i].CreatedAt = r.date
	}

	return resources, page, nil
}

func (r *resourceRepository) FindOne(context.Context, string) (*flare.Resource, error) {
	return nil, nil
}

func (r *resourceRepository) FindByURI(context.Context, string) (*flare.Resource, error) {
	return nil, nil
}

func (r *resourceRepository) Create(context.Context, *flare.Resource) error {
	return nil
}

func (r *resourceRepository) Delete(context.Context, string) error {
	return nil
}

func newResourceRepository(date time.Time, resources []flare.Resource) resourceRepository {
	base := memory.NewResource()

	for _, resource := range resources {
		if err := base.Create(context.Background(), &resource); err != nil {
			panic(err)
		}
	}

	return resourceRepository{base: base, date: date}
}
