// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Handler implements the HTTP handler to manage resources.
type Handler struct {
	repository      flare.ResourceRepositorier
	getResourceID   func(*http.Request) string
	getResourceURI  func(string) string
	parsePagination func(r *http.Request) (*flare.Pagination, error)
	writer          *infraHTTP.Writer
}

// Index receive the request to list the resources.
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

	re, rep, err := h.repository.Find(r.Context(), pag)
	if err != nil {
		h.writer.Error(w, "error during resources search", err, http.StatusInternalServerError)
		return
	}

	h.writer.Response(w, &response{
		Resources:  transformResources(re),
		Pagination: transformPagination(rep),
	}, http.StatusOK, nil)
}

// Show receive the request to show a resource.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	re, err := h.repository.FindByID(r.Context(), h.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		h.writer.Error(w, "error during resource search", err, status)
		return
	}

	h.writer.Response(w, transformResource(re), http.StatusOK, nil)
}

// Create receive the request to create a resource.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &resourceCreate{}
	)

	if err := d.Decode(content); err != nil {
		h.writer.Error(w, "error during body parse", err, http.StatusBadRequest)
		return
	}

	if err := content.valid(); err != nil {
		h.writer.Error(w, "invalid body content", err, http.StatusBadRequest)
		return
	}

	result := content.toFlareResource()
	if err := h.repository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.AlreadyExists() {
			status = http.StatusConflict
		}

		h.writer.Error(w, "error during resource create", err, status)
		return
	}

	header := make(http.Header)
	header.Set("Location", h.getResourceURI(result.ID))
	h.writer.Response(w, &response{Resource: transformResource(result)}, http.StatusCreated, header)
}

// Delete receive the request to delete a resource.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), h.getResourceID(r)); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		h.writer.Error(w, "error during resource delete", err, status)
		return
	}

	h.writer.Response(w, nil, http.StatusNoContent, nil)
}

// NewHandler initialize the service to handle HTTP requests.
func NewHandler(options ...func(*Handler)) (*Handler, error) {
	h := &Handler{}

	for _, option := range options {
		option(h)
	}

	if h.repository == nil {
		return nil, errors.New("repository not found")
	}

	if h.getResourceID == nil {
		return nil, errors.New("getResourceID not found")
	}

	if h.getResourceURI == nil {
		return nil, errors.New("getResourceURI not found")
	}

	if h.parsePagination == nil {
		return nil, errors.New("parsePagination not found")
	}

	if h.writer == nil {
		return nil, errors.New("writer not found")
	}

	return h, nil
}

// HandlerParsePagination set the function used to parse the pagination.
func HandlerParsePagination(fn func(r *http.Request) (*flare.Pagination, error)) func(*Handler) {
	return func(s *Handler) { s.parsePagination = fn }
}

// HandlerWriter set the function that return the content to client.
func HandlerWriter(writer *infraHTTP.Writer) func(*Handler) {
	return func(s *Handler) { s.writer = writer }
}

// HandlerRepository set the repository to access the resources.
func HandlerRepository(repo flare.ResourceRepositorier) func(*Handler) {
	return func(s *Handler) { s.repository = repo }
}

// HandlerGetResourceID set the function used to fetch the resourceID from the URL.
func HandlerGetResourceID(fn func(*http.Request) string) func(*Handler) {
	return func(s *Handler) { s.getResourceID = fn }
}

// HandlerGetResourceURI set the function used to generate the URI for a resource.
func HandlerGetResourceURI(fn func(string) string) func(*Handler) {
	return func(s *Handler) { s.getResourceURI = fn }
}
