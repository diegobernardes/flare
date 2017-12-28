// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Service implements the HTTP handler to manage documents.
type Service struct {
	documentRepository  flare.DocumentRepositorier
	resourceRepository  flare.ResourceRepositorier
	subscriptionTrigger flare.SubscriptionTrigger
	getDocumentID       func(*http.Request) string
	writer              *infraHTTP.Writer
}

// HandleShow receive the request to show a given document.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	d, err := s.documentRepository.FindOne(r.Context(), s.getDocumentID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}
		s.writer.Error(w, "error during document search", err, status)
		return
	}

	s.writer.Response(w, (*document)(d), http.StatusOK, nil)
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

	rawContent, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.writer.Error(
			w,
			"error during document process",
			errors.Wrap(err, "could not read the request body"),
			http.StatusInternalServerError,
		)
		return
	}

	if len(rawContent) == 0 {
		s.writer.Error(w, "missing document body", nil, http.StatusBadRequest)
		return
	}

	documentID := s.getDocumentID(r)
	resource := s.fetchResource(r.Context(), documentID, w)
	if resource == nil {
		return
	}

	doc, err := parseDocument(documentID, rawContent, resource)
	if err != nil {
		s.writer.Error(w, "error during document parse", err, http.StatusBadRequest)
		return
	}

	if err = doc.Valid(); err != nil {
		s.writer.Error(w, "invalid document", err, http.StatusBadRequest)
		return
	}

	if err = s.documentRepository.Update(r.Context(), doc); err != nil {
		s.writer.Error(w, "error during document update", err, http.StatusInternalServerError)
		return
	}

	if err := s.subscriptionTrigger.Update(r.Context(), doc); err != nil {
		s.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
		return
	}

	s.writer.Response(w, nil, http.StatusAccepted, nil)
}

// HandleDelete receive the request to delete a document.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		s.writer.Error(
			w,
			"error during document search",
			fmt.Errorf("query string not allowed '%s'", r.URL.RawQuery),
			http.StatusBadRequest,
		)
		return
	}

	documentID := s.getDocumentID(r)
	resource := s.fetchResource(r.Context(), documentID, w)
	if resource == nil {
		return
	}

	if err := s.subscriptionTrigger.Delete(r.Context(), &flare.Document{
		ID:        documentID,
		UpdatedAt: time.Now(),
		Resource:  *resource,
	}); err != nil {
		s.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
		return
	}

	s.writer.Response(w, nil, http.StatusAccepted, nil)
}

func (s *Service) fetchResource(
	ctx context.Context,
	documentID string,
	w http.ResponseWriter,
) *flare.Resource {
	doc, err := s.documentRepository.FindOne(ctx, documentID)
	if errRepo, ok := err.(flare.DocumentRepositoryError); ok && !errRepo.NotFound() {
		s.writer.Error(w, "error during document search", err, http.StatusInternalServerError)
		return nil
	}

	var resource *flare.Resource
	if doc == nil {
		resource, err = s.resourceRepository.FindByURI(ctx, documentID)
	} else {
		resource, err = s.resourceRepository.FindOne(ctx, doc.Resource.ID)
	}
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}
		s.writer.Error(w, "error during resource search", err, status)
		return nil
	}

	return resource
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

	if s.subscriptionTrigger == nil {
		return nil, errors.New("subscriptionTrigger not found")
	}

	if s.getDocumentID == nil {
		return nil, errors.New("getDocumentId not found")
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

// ServiceSubscriptionTrigger set the subscription trigger to process the document updates.
func ServiceSubscriptionTrigger(trigger flare.SubscriptionTrigger) func(*Service) {
	return func(s *Service) { s.subscriptionTrigger = trigger }
}

// ServiceGetDocumentID set the function to get the document id.
func ServiceGetDocumentID(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getDocumentID = fn }
}

// ServiceWriter set the writer to send the content to client.
func ServiceWriter(writer *infraHTTP.Writer) func(*Service) {
	return func(s *Service) { s.writer = writer }
}
