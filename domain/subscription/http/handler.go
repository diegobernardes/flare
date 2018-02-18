// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Handler implements the HTTP handler to manage subscriptions.
type Handler struct {
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	getResourceID          func(*http.Request) string
	getSubscriptionID      func(*http.Request) string
	getSubscriptionURI     func(string, string) string
	writer                 *infraHTTP.Writer
	parsePagination        func(r *http.Request) (*flare.Pagination, error)
}

// Index receive the request to list the subscriptions.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	pag, err := h.parsePagination(r)
	if err != nil {
		h.writer.Error(w, "error during pagination parse", err, http.StatusBadRequest)
		return
	}

	if err = pag.Valid(); err != nil {
		h.writer.Error(w, "invalid pagination", err, http.StatusBadRequest)
		return
	}

	subs, subsPag, err := h.subscriptionRepository.Find(r.Context(), pag, h.getResourceID(r))
	if err != nil {
		h.writer.Error(w, "error during subscriptions search", err, http.StatusInternalServerError)
		return
	}

	h.writer.Response(w, &response{
		Subscriptions: unmarshalSubscriptions(subs),
		Pagination:    unmarshalPagination(subsPag),
	}, http.StatusOK, nil)
}

// Show receive the request to show a subscription.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	subs, err := h.subscriptionRepository.FindByID(
		r.Context(), h.getResourceID(r), h.getSubscriptionID(r),
	)
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		h.writer.Error(w, "error during subscription search", err, status)
		return
	}

	h.writer.Response(w, unmarshalSubscription(subs), http.StatusOK, nil)
}

// Create receive the request to create a subscription.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	resource, err := h.resourceRepository.FindByID(r.Context(), h.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		h.writer.Error(w, "error during resource search", err, status)
		return
	}

	var sc subscriptionCreate
	if err = sc.parse(r.Body); err != nil {
		h.writer.Error(w, "error during body parse", err, http.StatusBadRequest)
		return
	}

	if err = sc.valid(resource); err != nil {
		h.writer.Error(w, "error during subscription create validation", err, http.StatusBadRequest)
		return
	}

	result, err := sc.marshal()
	if err != nil {
		h.writer.Error(w, "invalid subscription", err, http.StatusBadRequest)
		return
	}
	result.Resource.ID = resource.ID

	if err := h.subscriptionRepository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.AlreadyExists() {
			status = http.StatusConflict
		}

		h.writer.Error(w, "error during subscription create", err, status)
		return
	}

	header := make(http.Header)
	header.Set("Location", h.getSubscriptionURI(result.Resource.ID, result.ID))
	resp := &response{Subscription: unmarshalSubscription(result)}
	h.writer.Response(w, resp, http.StatusCreated, header)
}

// Delete receive the request to delete a subscription.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.subscriptionRepository.Delete(r.Context(), h.getResourceID(r), h.getSubscriptionID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		h.writer.Error(w, "error during subscription delete", err, status)
		return
	}

	h.writer.Response(w, nil, http.StatusNoContent, nil)
}

// NewHandler initialize the service to handle HTTP Requests.
func NewHandler(options ...func(*Handler)) (*Handler, error) {
	h := &Handler{}

	for _, option := range options {
		option(h)
	}

	if h.subscriptionRepository == nil {
		return nil, errors.New("subscriptionRepository not found")
	}

	if h.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if h.getResourceID == nil {
		return nil, errors.New("getResourceID not found")
	}

	if h.getSubscriptionID == nil {
		return nil, errors.New("getSubscriptionID not found")
	}

	if h.getSubscriptionURI == nil {
		return nil, errors.New("getSubscriptionURI not found")
	}

	if h.parsePagination == nil {
		return nil, errors.New("parsePagination not found")
	}

	if h.writer == nil {
		return nil, errors.New("writer not found")
	}

	return h, nil
}

// HandlerResourceRepository set the repository to access the resources.
func HandlerResourceRepository(repo flare.ResourceRepositorier) func(*Handler) {
	return func(s *Handler) { s.resourceRepository = repo }
}

// HandlerSubscriptionRepository set the repository to access the subscriptions.
func HandlerSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Handler) {
	return func(s *Handler) { s.subscriptionRepository = repo }
}

// HandlerGetResourceID the function to fetch the resourceId from the URL.
func HandlerGetResourceID(fn func(*http.Request) string) func(*Handler) {
	return func(s *Handler) { s.getResourceID = fn }
}

// HandlerGetSubscriptionID the function to fetch the subscriptionId from the URL.
func HandlerGetSubscriptionID(fn func(*http.Request) string) func(*Handler) {
	return func(s *Handler) { s.getSubscriptionID = fn }
}

// HandlerGetSubscriptionURI set the function to generate the URI or a given subscription.
func HandlerGetSubscriptionURI(fn func(string, string) string) func(*Handler) {
	return func(s *Handler) { s.getSubscriptionURI = fn }
}

// HandlerParsePagination set the function used to parse the pagination.
func HandlerParsePagination(fn func(r *http.Request) (*flare.Pagination, error)) func(*Handler) {
	return func(s *Handler) { s.parsePagination = fn }
}

// HandlerWriter set the function that return the content to client.
func HandlerWriter(writer *infraHTTP.Writer) func(*Handler) {
	return func(s *Handler) { s.writer = writer }
}
