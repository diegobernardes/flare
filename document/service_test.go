// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

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
	"github.com/diegobernardes/flare/repository/test"
	subscriptionTest "github.com/diegobernardes/flare/subscription/test"
)

func TestServiceHandleShow(t *testing.T) {
	tests := []struct {
		name       string
		req        *http.Request
		status     int
		header     http.Header
		body       []byte
		repository flare.DocumentRepositorier
	}{
		{
			"Not found",
			httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleShow.notFound.json"),
			test.NewDocument(),
		},
		{
			"Error at repository",
			httptest.NewRequest(http.MethodGet, "http://documents/123", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleShow.errorRepository.json"),
			test.NewDocument(test.DocumentError(errors.New("error at repository"))),
		},
		{
			"Found",
			httptest.NewRequest(http.MethodGet, "http://documents/456", nil),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleShow.found.json"),
			test.NewDocument(
				test.DocumentLoadSliceByteDocument(load("handleShow.foundInput.json")),
				test.DocumentDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
		},
	}

	for _, tt := range tests {
		trigger, err := subscriptionTest.NewTrigger()
		if err != nil {
			t.Error(errors.Wrap(err, "error during trigger initialization"))
			t.FailNow()
		}

		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionTrigger(trigger),
			ServiceDocumentRepository(tt.repository),
			ServiceResourceRepository(test.NewResource()),
			ServiceSubscriptionRepository(test.NewSubscription()),
			ServiceGetDocumentId(func(r *http.Request) string {
				return strings.Replace(r.URL.Path, "/", "", -1)
			}),
			ServiceGetDocumentURI(func(string) string {
				return ""
			}),
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
		name           string
		req            *http.Request
		status         int
		header         http.Header
		body           []byte
		repository     flare.DocumentRepositorier
		triggerOptions []func(*subscriptionTest.Trigger)
	}{
		{
			"Not found",
			httptest.NewRequest(http.MethodDelete, "http://documents/123", nil),
			http.StatusNotFound,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleDelete.notFound.json"),
			test.NewDocument(),
			nil,
		},
		{
			"Repository error",
			httptest.NewRequest(http.MethodDelete, "http://documents/123", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleDelete.repositoryErr.json"),
			test.NewDocument(test.DocumentError(errors.New("repository error"))),
			nil,
		},
		{
			"Error during delete",
			httptest.NewRequest(http.MethodDelete, "http://documents/456", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleDelete.deleteErr.json"),
			test.NewDocument(
				test.DocumentDeleteError(errors.New("repository error")),
				test.DocumentLoadSliceByteDocument(load("handleShow.foundInput.json")),
			),
			nil,
		},
		{
			"Error during subscription trigger",
			httptest.NewRequest(http.MethodDelete, "http://documents/456", nil),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleDelete.triggerErr.json"),
			test.NewDocument(test.DocumentLoadSliceByteDocument(load("handleShow.foundInput.json"))),
			[]func(*subscriptionTest.Trigger){
				subscriptionTest.TriggerError(errors.New("error during trigger")),
			},
		},
		{
			"Success",
			httptest.NewRequest(http.MethodDelete, "http://documents/456", nil),
			http.StatusNoContent,
			http.Header{},
			nil,
			test.NewDocument(test.DocumentLoadSliceByteDocument(load("handleShow.foundInput.json"))),
			nil,
		},
	}

	for _, tt := range tests {
		trigger, err := subscriptionTest.NewTrigger(tt.triggerOptions...)
		if err != nil {
			t.Error(errors.Wrap(err, "error during trigger initialization"))
			t.FailNow()
		}

		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionTrigger(trigger),
			ServiceDocumentRepository(tt.repository),
			ServiceResourceRepository(test.NewResource()),
			ServiceSubscriptionRepository(test.NewSubscription()),
			ServiceGetDocumentId(func(r *http.Request) string {
				return strings.Replace(r.URL.Path, "/", "", -1)
			}),
			ServiceGetDocumentURI(func(string) string {
				return ""
			}),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleDelete, tt.req, tt.body))
	}
}

func TestServiceHandleUpdate(t *testing.T) {
	tests := []struct {
		name                   string
		req                    *http.Request
		status                 int
		header                 http.Header
		body                   []byte
		documentRepository     flare.DocumentRepositorier
		resourceRepository     flare.ResourceRepositorier
		subscriptionRepository flare.SubscriptionRepositorier
		triggerOptions         []func(*subscriptionTest.Trigger)
	}{
		{
			"Invalid document",
			httptest.NewRequest(http.MethodPut, "http://documents/app.com/123", nil),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.documentInvalid.json"),
			test.NewDocument(),
			test.NewResource(),
			test.NewSubscription(),
			nil,
		},
		{
			"Error while search document by uri",
			httptest.NewRequest(http.MethodPut, "http://documents/app.com/123", bytes.NewBufferString("{}")),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.resourceErr.json"),
			test.NewDocument(),
			test.NewResource(test.ResourceFindByURIError(errors.New("error on repository"))),
			test.NewSubscription(),
			nil,
		},
		{
			"Document not valid",
			httptest.NewRequest(http.MethodPut, "http://documents/app.com/123", bytes.NewBufferString("{}")),
			http.StatusBadRequest,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.documentInvalid2.json"),
			test.NewDocument(),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(),
			nil,
		},
		{
			"Error while find reference document",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.documentErr.json"),
			test.NewDocument(
				test.DocumentFindOneError(errors.New("error on repository")),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(),
			nil,
		},
		{
			"Error while find search of subscription",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.hasSubscriptionErr.json"),
			test.NewDocument(),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(
				test.SubscriptionHasSubscriptionError(errors.New("error on repository")),
			),
			nil,
		},
		{
			"Updated the document without subscription",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusOK,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.successWithoutSubscription.json"),
			test.NewDocument(),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(),
			nil,
		},
		{
			"Document update without subscription",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusCreated,
			http.Header{
				"Content-Type": []string{"application/json"},
				"Location":     []string{"http://documents/app.com/123"},
			},
			load("handleUpdate.create1.json"),
			test.NewDocument(
				test.DocumentDate(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(
				test.SubscriptionLoadSliceByteSubscription(load("handleUpdate.subscriptionInput.json")),
			),
			nil,
		},
		{
			"Error on document update",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.updateError.json"),
			test.NewDocument(
				test.DocumentUpdateError(errors.New("error on repository")),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(
				test.SubscriptionLoadSliceByteSubscription(load("handleUpdate.subscriptionInput.json")),
			),
			nil,
		},
		{
			"Error on document update with reference document",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusInternalServerError,
			http.Header{"Content-Type": []string{"application/json"}},
			load("handleUpdate.updateError.json"),
			test.NewDocument(
				test.DocumentLoadSliceByteDocument(load("handleUpdate.referenceDocument.json")),
				test.DocumentUpdateError(errors.New("error on repository")),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(
				test.SubscriptionLoadSliceByteSubscription(load("handleUpdate.subscriptionInput.json")),
			),
			nil,
		},
		{
			"Skip older document",
			httptest.NewRequest(
				http.MethodPut,
				"http://documents/app.com/123",
				bytes.NewBuffer(load("handleUpdate.validDocument.json")),
			),
			http.StatusOK,
			http.Header{
				"Content-Type": {"application/json"},
				"Location":     {"http://documents/app.com/123"},
			},
			load("handleUpdate.skipedDocument.json"),
			test.NewDocument(
				test.DocumentLoadSliceByteDocument(load("handleUpdate.referenceDocument2.json")),
			),
			test.NewResource(
				test.ResourceLoadSliceByteResource(load("handleUpdate.resourceCreate.json")),
			),
			test.NewSubscription(
				test.SubscriptionLoadSliceByteSubscription(load("handleUpdate.subscriptionInput.json")),
			),
			nil,
		},
	}

	for _, tt := range tests {
		trigger, err := subscriptionTest.NewTrigger(tt.triggerOptions...)
		if err != nil {
			t.Error(errors.Wrap(err, "error during trigger initialization"))
			t.FailNow()
		}

		service, err := NewService(
			ServiceLogger(log.NewNopLogger()),
			ServiceSubscriptionTrigger(trigger),
			ServiceDocumentRepository(tt.documentRepository),
			ServiceResourceRepository(tt.resourceRepository),
			ServiceSubscriptionRepository(tt.subscriptionRepository),
			ServiceGetDocumentId(func(r *http.Request) string {
				return strings.Replace(r.URL.String(), "http://documents/", "", -1)
			}),
			ServiceGetDocumentURI(func(id string) string { return fmt.Sprintf("http://documents/%s", id) }),
		)
		if err != nil {
			t.Error(errors.Wrap(err, "error during service initialization"))
			t.FailNow()
		}

		t.Run(tt.name, infraHTTP.Handler(tt.status, tt.header, service.HandleUpdate, tt.req, tt.body))
	}
}

func TestNewService(t *testing.T) {
	trigger := func(t *testing.T) flare.SubscriptionTrigger {
		trigger, err := subscriptionTest.NewTrigger()
		if err != nil {
			t.Error(errors.Wrap(err, "error during trigger initialization"))
			t.FailNow()
		}
		return trigger
	}

	tests := []struct {
		name     string
		options  []func(*Service)
		hasError bool
		fn       func(t *testing.T) flare.SubscriptionTrigger
	}{
		{
			"Mising logger",
			[]func(*Service){},
			true,
			nil,
		},
		{
			"Mising subscription trigger",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
			},
			true,
			nil,
		},
		{
			"Mising document repository",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
			},
			true,
			trigger,
		},
		{
			"Mising resource repository",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceDocumentRepository(test.NewDocument()),
			},
			true,
			trigger,
		},
		{
			"Mising subscription repository",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceDocumentRepository(test.NewDocument()),
				ServiceResourceRepository(test.NewResource()),
			},
			true,
			trigger,
		},
		{
			"Mising getDocumentId repository",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceDocumentRepository(test.NewDocument()),
				ServiceResourceRepository(test.NewResource()),
				ServiceSubscriptionRepository(test.NewSubscription()),
			},
			true,
			trigger,
		},
		{
			"Mising getDocumentURI repository",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceDocumentRepository(test.NewDocument()),
				ServiceResourceRepository(test.NewResource()),
				ServiceSubscriptionRepository(test.NewSubscription()),
				ServiceGetDocumentId(func(*http.Request) string { return "" }),
			},
			true,
			trigger,
		},
		{
			"Success",
			[]func(*Service){
				ServiceLogger(log.NewNopLogger()),
				ServiceDocumentRepository(test.NewDocument()),
				ServiceResourceRepository(test.NewResource()),
				ServiceSubscriptionRepository(test.NewSubscription()),
				ServiceGetDocumentId(func(*http.Request) string { return "" }),
				ServiceGetDocumentURI(func(string) string { return "" }),
			},
			false,
			trigger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fn != nil {
				tt.options = append(tt.options, ServiceSubscriptionTrigger(tt.fn(t)))
			}

			_, err := NewService(tt.options...)
			if tt.hasError != (err != nil) {
				t.Errorf("NewService invalid result, want '%v', got '%v'", tt.hasError, err)
			}
		})
	}
}
