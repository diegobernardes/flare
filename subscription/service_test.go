// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http/test"
	"github.com/diegobernardes/flare/repository/memory"
	"github.com/diegobernardes/flare/repository/test"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name     string
		options  []func(*Service)
		hasError bool
	}{
		{
			"Fail",
			[]func(*Service){},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(-1),
			},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
			},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
			},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
			},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceId(func(*http.Request) string { return "" }),
			},
			true,
		},
		{
			"Fail",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceId(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionId(func(*http.Request) string { return "" }),
			},
			true,
		},
		{
			"Success",
			[]func(*Service){
				ServiceDefaultLimit(1),
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceId(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionId(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionURI(func(string, string) string { return "" }),
			},
			false,
		},
		{
			"Success",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceSubscriptionRepository(memory.NewSubscription()),
				ServiceResourceRepository(memory.NewResource()),
				ServiceGetResourceId(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionId(func(*http.Request) string { return "" }),
				ServiceGetSubscriptionURI(func(string, string) string { return "" }),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.options...)
			if tt.hasError != (err != nil) {
				t.Errorf("NewService invalid result, want '%v', got '%v'", tt.hasError, err)
			}
		})
	}
}

func TestServiceHandleIndex(t *testing.T) {
	tests := []struct {
		name                   string
		req                    *http.Request
		status                 int
		header                 http.Header
		body                   []byte
		subscriptionRepository flare.SubscriptionRepositorier
		resourceRepository     flare.ResourceRepositorier
	}{
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources/123/subscriptions?limit=sample", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.invalidPaginationType1.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=sample", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.invalidPaginationType2.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources/123/subscriptions?limit=-1", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.invalidPaginationValue1.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Invalid pagination",
			httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=-1", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.invalidPaginationValue2.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Error during resource search",
			httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.resourceErrorSearch.json"),
			test.NewSubscription(),
			test.NewResource(test.ResourceError(errors.New("error during repository search"))),
		},
		{
			"Resource not found",
			httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.resourceNotFound.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Error during subscription search",
			httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.subscriptionErrorSearch.json"),
			test.NewSubscription(test.SubscriptionError(
				errors.New("error during repository search"),
			)),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleInput.inputResource.json")),
				test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
		},
		{
			"Empty search",
			httptest.NewRequest("GET", "http://resources/123/subscriptions", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.emptySearch.json"),
			test.NewSubscription(
				test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleInput.inputResource.json")),
				test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
		},
		{
			"Results with pagination",
			httptest.NewRequest("GET", "http://resources/123/subscriptions?offset=1", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleIndex.resultsWithPagination.json"),
			test.NewSubscription(
				test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				test.SubscriptionLoadSliceByteSubscription(load("handleIndex.input.json")),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleInput.inputResource.json")),
				test.ResourceDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
		},
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceDefaultLimit(30),
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionRepository(tt.subscriptionRepository),
			ServiceResourceRepository(tt.resourceRepository),
			ServiceGetResourceId(func(r *http.Request) string {
				id := strings.Replace(r.URL.Path, "/subscriptions", "", -1)
				id = strings.Replace(id, "/", "", -1)
				return id
			}),
			ServiceGetSubscriptionId(func(r *http.Request) string {
				return strings.Replace(r.URL.String(), "http://resources/", "", -1)
			}),
			ServiceGetSubscriptionURI(func(string, string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleIndex, tt.req, tt.body))
	}
}

func TestServiceHandleShow(t *testing.T) {
	tests := []struct {
		name                   string
		req                    *http.Request
		status                 int
		header                 http.Header
		body                   []byte
		subscriptionRepository flare.SubscriptionRepositorier
		resourceRepository     flare.ResourceRepositorier
	}{
		{
			"Not found",
			httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleShow.notFound.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Found found",
			httptest.NewRequest("GET", "http://resources/123/subscriptions/456", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleShow.foundOutput.json"),
			test.NewSubscription(
				test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				test.SubscriptionLoadSliceByteSubscription(load("handleShow.foundInput.json")),
			),
			test.NewResource(),
		},
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionRepository(tt.subscriptionRepository),
			ServiceResourceRepository(tt.resourceRepository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[1]
			}),
			ServiceGetSubscriptionId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[3]
			}),
			ServiceGetSubscriptionURI(func(string, string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleShow, tt.req, tt.body))
	}
}

func TestServiceHandleDelete(t *testing.T) {
	tests := []struct {
		name                   string
		req                    *http.Request
		status                 int
		header                 http.Header
		body                   []byte
		subscriptionRepository flare.SubscriptionRepositorier
		resourceRepository     flare.ResourceRepositorier
	}{
		{
			"Not found",
			httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleDelete.notFound.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Delete",
			httptest.NewRequest(http.MethodDelete, "http://resources/123/subscriptions/456", nil),
			http.StatusNoContent,
			http.Header{},
			nil,
			test.NewSubscription(
				test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
				test.SubscriptionLoadSliceByteSubscription(load("handleShow.foundInput.json")),
			),
			test.NewResource(),
		},
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionRepository(tt.subscriptionRepository),
			ServiceResourceRepository(tt.resourceRepository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[1]
			}),
			ServiceGetSubscriptionId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[3]
			}),
			ServiceGetSubscriptionURI(func(string, string) string { return "" }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleDelete, tt.req, tt.body))
	}
}

func TestServiceHandleCreate(t *testing.T) {
	tests := []struct {
		name                   string
		req                    *http.Request
		status                 int
		header                 http.Header
		body                   []byte
		subscriptionRepository flare.SubscriptionRepositorier
		resourceRepository     flare.ResourceRepositorier
	}{
		{
			"Invalid Content",
			httptest.NewRequest(http.MethodPost, "http://resources/123/subscriptions", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleCreate.invalidContent1.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Invalid Content",
			httptest.NewRequest(
				http.MethodPost, "http://resources/123/subscriptions", bytes.NewBufferString("{}"),
			),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleCreate.invalidContent2.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Invalid Content",
			httptest.NewRequest(
				http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
					load("handleCreate.invalidInput.json"),
				),
			),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleCreate.invalidContent3.json"),
			test.NewSubscription(),
			test.NewResource(),
		},
		{
			"Error at repository",
			httptest.NewRequest(
				http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
					load("handleCreate.input.json"),
				),
			),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleCreate.errorRepository.json"),
			test.NewSubscription(
				test.SubscriptionError(errors.New("error at repository")),
			),
			test.NewResource(),
		},
		{
			"Conflict at repository",
			httptest.NewRequest(
				http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
					load("handleCreate.input.json"),
				),
			),
			http.StatusConflict,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleCreate.repositoryConflict.json"),
			test.NewSubscription(
				test.SubscriptionCreateId("456"),
				test.SubscriptionLoadSliceByteSubscription(load("handleCreate.inputArray.json")),
			),
			test.NewResource(),
		},
		{
			"Create",
			httptest.NewRequest(
				http.MethodPost, "http://resources/123/subscriptions", bytes.NewBuffer(
					load("handleCreate.input.json"),
				),
			),
			http.StatusCreated,
			http.Header{
				"Content-Type": []string{"application/json"},
				"Location":     []string{"http://resources/123/subscriptions/456"},
			},
			load("handleCreate.create.json"),
			test.NewSubscription(
				test.SubscriptionCreateId("456"),
				test.SubscriptionDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
			test.NewResource(),
		},
	}

	for _, tt := range tests {
		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionRepository(tt.subscriptionRepository),
			ServiceResourceRepository(tt.resourceRepository),
			ServiceGetResourceId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[1]
			}),
			ServiceGetSubscriptionId(func(r *http.Request) string {
				return strings.Split(r.URL.Path, "/")[3]
			}),
			ServiceGetSubscriptionURI(func(resourceId string, id string) string {
				return fmt.Sprintf("http://resources/%s/subscriptions/%s", resourceId, id)
			}),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleCreate, tt.req, tt.body))
	}
}
