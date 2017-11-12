// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Service implements the HTTP handler to manage resources.
type Service struct {
	repository      flare.ResourceRepositorier
	getResourceID   func(*http.Request) string
	getResourceURI  func(string) string
	writeResponse   func(http.ResponseWriter, interface{}, int, http.Header)
	parsePagination func(r *http.Request) (*flare.Pagination, error)
}

// HandleIndex receive the request to list the resources.
func (s *Service) HandleIndex(w http.ResponseWriter, r *http.Request) {
	pag, err := s.parsePagination(r)
	if err != nil {
		s.writeError(w, err, "error during pagination parse", http.StatusBadRequest)
		return
	}

	if err = pag.Valid(); err != nil {
		s.writeError(w, err, "invalid pagination", http.StatusBadRequest)
		return
	}

	re, rep, err := s.repository.FindAll(r.Context(), pag)
	if err != nil {
		s.writeError(w, err, "error during resources search", http.StatusInternalServerError)
		return
	}

	s.writeResponse(w, &response{
		Resources:  transformResources(re),
		Pagination: transformPagination(rep),
	}, http.StatusOK, nil)
}

// HandleShow receive the request to show a resource.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	re, err := s.repository.FindOne(r.Context(), s.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during resource search", status)
		return
	}

	s.writeResponse(w, transformResource(re), http.StatusOK, nil)
}

// HandleCreate receive the request to create a resource.
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &resourceCreate{}
	)

	if err := d.Decode(content); err != nil {
		s.writeError(w, err, "error during body parse", http.StatusBadRequest)
		return
	}

	if err := content.valid(); err != nil {
		s.writeError(w, err, "invalid body content", http.StatusBadRequest)
		return
	}

	result := content.toFlareResource()
	if err := s.repository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok {
			if errRepo.PathConflict() || errRepo.AlreadyExists() {
				status = http.StatusConflict
			}
		}

		s.writeError(w, err, "error during resource create", status)
		return
	}

	header := make(http.Header)
	header.Set("Location", s.getResourceURI(result.Id))
	s.writeResponse(w, &response{Resource: transformResource(result)}, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a resource.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.repository.Delete(r.Context(), s.getResourceID(r)); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during resource delete", status)
		return
	}

	s.writeResponse(w, nil, http.StatusNoContent, nil)
}

// NewService initialize the service to handle HTTP requests.
func NewService(options ...func(*Service)) (*Service, error) {
	service := &Service{}

	for _, option := range options {
		option(service)
	}

	if service.repository == nil {
		return nil, errors.New("repository not found")
	}

	if service.getResourceID == nil {
		return nil, errors.New("getResourceID not found")
	}

	if service.getResourceURI == nil {
		return nil, errors.New("getResourceURI not found")
	}

	if service.parsePagination == nil {
		return nil, errors.New("parsePagination not found")
	}

	if service.writeResponse == nil {
		return nil, errors.New("writeResponse not found")
	}

	return service, nil
}

// ServiceParsePagination set the function used to parse the pagination.
func ServiceParsePagination(fn func(r *http.Request) (*flare.Pagination, error)) func(*Service) {
	return func(s *Service) {
		s.parsePagination = fn
	}
}

// ServiceWriteResponse set the function that return the content to client.
func ServiceWriteResponse(
	fn func(http.ResponseWriter, interface{}, int, http.Header),
) func(*Service) {
	return func(s *Service) {
		s.writeResponse = fn
	}
}

// ServiceRepository set the repository to access the resources.
func ServiceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.repository = repo }
}

// ServiceGetResourceID set the function used to fetch the resourceID from the URL.
func ServiceGetResourceID(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getResourceID = fn }
}

// ServiceGetResourceURI set the function used to generate the URI for a resource.
func ServiceGetResourceURI(fn func(string) string) func(*Service) {
	return func(s *Service) { s.getResourceURI = fn }
}
