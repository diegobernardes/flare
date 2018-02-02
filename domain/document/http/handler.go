// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Handler implements the HTTP handler to manage documents.
type Handler struct {
	documentRepository  flare.DocumentRepositorier
	resourceRepository  flare.ResourceRepositorier
	subscriptionTrigger flare.SubscriptionTrigger
	getDocumentID       func(*http.Request) string
	writer              *infraHTTP.Writer
}

// Show receive the request to show a given document.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	d, err := h.documentRepository.FindByID(r.Context(), h.getDocumentID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}
		h.writer.Error(w, "error during document search", err, status)
		return
	}

	h.writer.Response(w, (*document)(d), http.StatusOK, nil)
}

// Update process the request to update a document.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		h.writer.Error(
			w,
			"error during document update",
			fmt.Errorf("query string not allowed '%s'", r.URL.RawQuery),
			http.StatusBadRequest,
		)
		return
	}

	documentID := h.getDocumentID(r)
	resource := h.fetchResource(r.Context(), documentID, w)
	if resource == nil {
		return
	}

	doc, err := parseDocument(r.Body, documentID, resource)
	if err != nil {
		h.writer.Error(w, "error during document parse", err, http.StatusBadRequest)
		return
	}

	if err = h.documentRepository.Update(r.Context(), doc); err != nil {
		h.writer.Error(w, "error during document update", err, http.StatusInternalServerError)
		return
	}

	err = h.subscriptionTrigger.Push(r.Context(), doc, flare.SubscriptionTriggerUpdate)
	if err != nil {
		h.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
		return
	}

	h.writer.Response(w, nil, http.StatusAccepted, nil)
}

// Delete receive the request to delete a document.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		h.writer.Error(
			w,
			"error during document delete",
			fmt.Errorf("query string not allowed '%s'", r.URL.RawQuery),
			http.StatusBadRequest,
		)
		return
	}

	documentID := h.getDocumentID(r)
	resource := h.fetchResource(r.Context(), documentID, w)
	if resource == nil {
		return
	}

	if err := h.subscriptionTrigger.Push(r.Context(), &flare.Document{
		ID:        documentID,
		UpdatedAt: time.Now(),
		Resource:  *resource,
	}, flare.SubscriptionTriggerDelete); err != nil {
		h.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
		return
	}

	h.writer.Response(w, nil, http.StatusAccepted, nil)
}

func (h *Handler) fetchResource(
	ctx context.Context,
	documentID string,
	w http.ResponseWriter,
) *flare.Resource {
	doc, err := h.documentRepository.FindByID(ctx, documentID)
	if err != nil {
		if errRepo := err.(flare.DocumentRepositoryError); !errRepo.NotFound() {
			h.writer.Error(w, "error during document search", err, http.StatusInternalServerError)
			return nil
		}
	}

	var resource *flare.Resource
	if doc == nil {
		resource, err = h.resourceRepository.FindByURI(ctx, documentID)
	} else {
		resource, err = h.resourceRepository.FindByID(ctx, doc.Resource.ID)
	}
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}
		h.writer.Error(w, "error during resource search", err, status)
		return nil
	}

	return resource
}

// NewHandler initialize the service to handle HTTP requests.
func NewHandler(options ...func(*Handler)) (*Handler, error) {
	h := &Handler{}

	for _, option := range options {
		option(h)
	}

	if h.documentRepository == nil {
		return nil, errors.New("documentRepository not found")
	}

	if h.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if h.subscriptionTrigger == nil {
		return nil, errors.New("subscriptionTrigger not found")
	}

	if h.getDocumentID == nil {
		return nil, errors.New("getDocumentId not found")
	}

	if h.writer == nil {
		return nil, errors.New("writer not Found")
	}

	return h, nil
}

// HandlerDocumentRepository set the repository to access the documents.
func HandlerDocumentRepository(repo flare.DocumentRepositorier) func(*Handler) {
	return func(s *Handler) { s.documentRepository = repo }
}

// HandlerResourceRepository set the repository to access the resources.
func HandlerResourceRepository(repo flare.ResourceRepositorier) func(*Handler) {
	return func(s *Handler) { s.resourceRepository = repo }
}

// HandlerSubscriptionTrigger set the subscription trigger to process the document updates.
func HandlerSubscriptionTrigger(trigger flare.SubscriptionTrigger) func(*Handler) {
	return func(s *Handler) { s.subscriptionTrigger = trigger }
}

// HandlerGetDocumentID set the function to get the document id.
func HandlerGetDocumentID(fn func(*http.Request) string) func(*Handler) {
	return func(s *Handler) { s.getDocumentID = fn }
}

// HandlerWriter set the writer to send the content to client.
func HandlerWriter(writer *infraHTTP.Writer) func(*Handler) {
	return func(s *Handler) { s.writer = writer }
}
