// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Handler implements the HTTP handler to manage documents.
type Handler struct {
	documentRepository     flare.DocumentRepositorier
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	subscriptionTrigger    flare.SubscriptionTrigger
	getDocumentID          func(*http.Request) string
	writer                 *infraHTTP.Writer
}

// Show receive the request to show a given document.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id := h.parseID(w, r)
	if id == nil {
		return
	}

	document, err := h.documentRepository.FindByID(r.Context(), *id)
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}
		h.writer.Error(w, "error during document search", err, status)
		return
	}

	h.writer.Response(w, unmarshal(document), http.StatusOK, nil)
}

// Update process the request to update a document.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := h.parseID(w, r)
	if id == nil {
		return
	}

	resource := h.fetchResource(r.Context(), id, w)
	if resource == nil {
		return
	}

	_, pagination, err := h.subscriptionRepository.Find(
		r.Context(), &flare.Pagination{Limit: 1}, resource.ID,
	)
	if err != nil {
		h.writer.Error(
			w,
			"error during check if the resource has any subscription",
			err,
			http.StatusInternalServerError,
		)
		return
	}
	if pagination.Total == 0 {
		h.writer.Response(w, nil, http.StatusAccepted, nil)
		return
	}

	rawDoc := &document{
		ID:        *id,
		Resource:  *resource,
		UpdatedAt: time.Now(),
	}

	if err = rawDoc.parseBody(r.Body); err != nil {
		h.writer.Error(w, "invalid body", err, http.StatusBadRequest)
		return
	}

	if err = rawDoc.parseRevision(); err != nil {
		h.writer.Error(w, "invalid body", err, http.StatusBadRequest)
		return
	}

	doc := marshal(rawDoc)
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
	id := h.parseID(w, r)
	if id == nil {
		return
	}

	resource := h.fetchResource(r.Context(), id, w)
	if resource == nil {
		return
	}

	doc := &flare.Document{
		ID:        *id,
		UpdatedAt: time.Now(),
		Resource:  *resource,
	}
	action := flare.SubscriptionTriggerDelete

	if err := h.subscriptionTrigger.Push(r.Context(), doc, action); err != nil {
		h.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
		return
	}

	h.writer.Response(w, nil, http.StatusAccepted, nil)
}

func (h *Handler) fetchResource(
	ctx context.Context, id *url.URL, w http.ResponseWriter,
) *flare.Resource {
	doc, err := h.documentRepository.FindByID(ctx, *id)
	if err != nil {
		if errRepo := err.(flare.DocumentRepositoryError); !errRepo.NotFound() {
			h.writer.Error(w, "error during document search", err, http.StatusInternalServerError)
			return nil
		}
	}

	var resource *flare.Resource
	if doc == nil {
		resource, err = h.resourceRepository.FindByURI(ctx, *id)
	} else {
		resource, err = h.resourceRepository.FindByID(ctx, doc.Resource.ID)
		if err != nil {
			if errRepo := err.(flare.ResourceRepositoryError); errRepo.NotFound() {
				resource, err = h.resourceRepository.FindByURI(ctx, *id)
			}
		}
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

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) *url.URL {
	id, err := url.Parse(h.getDocumentID(r))
	if err != nil {
		h.writer.Error(w, "error during id parse", err, http.StatusBadRequest)
		return nil
	}

	if err := validEndpoint(id); err != nil {
		h.writer.Error(w, "invalid id", err, http.StatusBadRequest)
		return nil
	}

	return id
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

	if h.subscriptionRepository == nil {
		return nil, errors.New("subscriptionRepository not found")
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

// HandlerSubscriptionRepository set the repository to access the subscriptions.
func HandlerSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Handler) {
	return func(s *Handler) { s.subscriptionRepository = repo }
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
