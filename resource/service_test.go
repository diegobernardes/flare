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
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
)

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
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceRepository(&tt.repository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Replace(r.URL.String(), "http://resources/", "", -1)
			}),
			ServiceGetResourceURI(func(string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
		}

		t.Run(tt.name, testService(tt.status, tt.header, service.HandleIndex, tt.req, tt.body))
	}
}

func TestHandleShow(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		status     int
		header     http.Header
		body       []byte
		repository resourceRepository
	}{
		{
			"Resource not found",
			httptest.NewRequest("GET", "http://resources/123", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("showNotFound.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{},
			),
		},
		{
			"Resource not found",
			httptest.NewRequest("GET", "http://resources/123", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("showRepositoryError.json"),
			resourceRepository{err: errors.New("error during repository search")},
		},
		{
			"Valid resource",
			httptest.NewRequest("GET", "http://resources/123", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("showSuccess.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{
					{
						Id:        "123",
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
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceRepository(&tt.repository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Replace(r.URL.String(), "http://resources/", "", -1)
			}),
			ServiceGetResourceURI(func(string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
		}

		t.Run(tt.name, testService(tt.status, tt.header, service.HandleShow, tt.req, tt.body))
	}
}

func TestHandleDelete(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		status     int
		header     http.Header
		body       []byte
		repository resourceRepository
	}{
		{
			"Resource not found",
			httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("deleteNotFound.json"),
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{},
			),
		},
		{
			"Error during delete",
			httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("deleteError.json"),
			resourceRepository{err: errors.New("error during repository delete")},
		},
		{
			"Delete with success",
			httptest.NewRequest(http.MethodDelete, "http://resources/123", nil),
			http.StatusNoContent,
			http.Header{},
			nil,
			newResourceRepository(
				time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				[]flare.Resource{
					{
						Id:        "123",
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
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceRepository(&tt.repository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Replace(r.URL.String(), "http://resources/", "", -1)
			}),
			ServiceGetResourceURI(func(string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
		}

		t.Run(tt.name, testService(tt.status, tt.header, service.HandleDelete, tt.req, tt.body))
	}
}

func testService(
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
			t.Errorf("status, want '%v', got '%v'", status, resp.Status)
		}

		if !reflect.DeepEqual(header, resp.Header) {
			t.Errorf("status, want '%v', got '%v'", header, resp.Header)
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
			t.Errorf("body, want '%v', got '%v'", b1, b2)
		}
	}
}

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

func (r *resourceRepository) FindOne(ctx context.Context, id string) (*flare.Resource, error) {
	if r.err != nil {
		return nil, r.err
	}

	res, err := r.base.FindOne(ctx, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = r.date

	return res, nil
}

func (r *resourceRepository) FindByURI(context.Context, string) (*flare.Resource, error) {
	return nil, nil
}

func (r *resourceRepository) Create(context.Context, *flare.Resource) error {
	return nil
}

func (r *resourceRepository) Delete(ctx context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	return r.base.Delete(ctx, id)
}

func newResourceRepository(date time.Time, resources []flare.Resource) resourceRepository {
	base := memory.NewResource(
		memory.ResourceSubscriptionRepository(memory.NewSubscription()),
	)

	for _, resource := range resources {
		if err := base.Create(context.Background(), &resource); err != nil {
			panic(err)
		}
	}

	return resourceRepository{base: base, date: date}
}
