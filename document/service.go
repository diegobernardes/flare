// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Service implements the HTTP handler to manage documents.
type Service struct {
	documentRepository flare.DocumentRepositorier
	resourceRepository flare.ResourceRepositorier
	getDocumentId      func(*http.Request) string
	worker             *Worker
	writer             *infraHTTP.Writer
}

// HandleShow receive the request to show a given document.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	d, err := s.documentRepository.FindOne(r.Context(), s.getDocumentId(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writer.Error(w, "error during document search", err, status)
		return
	}

	s.writer.Response(w, transformDocument(d), http.StatusOK, nil)
}

// HandleUpdate process the request to update a document.
func (s *Service) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		s.writer.Error(
			w,
			"error during document search",
			fmt.Errorf("query string not allowed '%s'", r.URL.RawQuery),
			http.StatusBadRequest,
		)
		return
	}

	if r.URL.Fragment != "" {
		s.writer.Error(
			w,
			"error during document search",
			fmt.Errorf("fragment not allowed '%s'", r.URL.Fragment),
			http.StatusBadRequest,
		)
		return
	}

	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.writer.Error(
			w,
			"error during document process",
			errors.Wrap(err, "could not read the request body"),
			http.StatusInternalServerError,
		)
		return
	}

	err = s.worker.Push(r.Context(), s.getDocumentId(r), flare.SubscriptionTriggerUpdate, content)
	if err != nil {
		s.writer.Error(
			w,
			"error during document process",
			errors.Wrap(err, "could not push the document to worker"),
			http.StatusInternalServerError,
		)
		return
	}

	s.writer.Response(w, nil, http.StatusAccepted, nil)
}

// HandleDelete receive the request to delete a document.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	err := s.worker.Push(r.Context(), s.getDocumentId(r), flare.SubscriptionTriggerDelete, nil)
	if err != nil {
		s.writer.Error(w, "error during document delete", err, http.StatusInternalServerError)
		return
	}

	s.writer.Response(w, nil, http.StatusAccepted, nil)
}

// NewService initialize the service to handle HTTP requests.
func NewService(options ...func(*Service)) (*Service, error) {
	s := &Service{}

	for _, option := range options {
		option(s)
	}

	if s.documentRepository == nil {
		return nil, errors.New("documentRepository not found")
	}

	if s.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if s.getDocumentId == nil {
		return nil, errors.New("getDocumentId not found")
	}

	if s.worker == nil {
		return nil, errors.New("worker not found")
	}

	if s.writer == nil {
		return nil, errors.New("writer not Found")
	}

	return s, nil
}

// ServiceDocumentRepository set the repository to access the documents.
func ServiceDocumentRepository(repo flare.DocumentRepositorier) func(*Service) {
	return func(s *Service) { s.documentRepository = repo }
}

// ServiceResourceRepository set the repository to access the resources.
func ServiceResourceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.resourceRepository = repo }
}

// ServiceGetDocumentId set the function to get the document id.
func ServiceGetDocumentId(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getDocumentId = fn }
}

// ServiceWorker set the worker to enqueue the messages to be processed async.
func ServiceWorker(worker *Worker) func(*Service) {
	return func(s *Service) { s.worker = worker }
}

// ServiceWriter set the writer to send the content to client.
func ServiceWriter(writer *infraHTTP.Writer) func(*Service) {
	return func(s *Service) { s.writer = writer }
}
