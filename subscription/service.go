// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/diegobernardes/flare"
)

// Service implements the HTTP handler to manage subscriptions.
type Service struct {
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	getResourceID          func(*http.Request) string
	getSubscriptionID      func(*http.Request) string
	getSubscriptionURI     func(string, string) string
	writeResponse          func(http.ResponseWriter, interface{}, int, http.Header)
	parsePagination        func(r *http.Request) (*flare.Pagination, error)
}

// HandleIndex receive the request to list the subscriptions.
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

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during resource search", status)
		return
	}

	subs, subsPag, err := s.subscriptionRepository.FindAll(
		r.Context(), pag, resource.ID,
	)
	if err != nil {
		s.writeError(w, err, "error during subscriptions search", http.StatusInternalServerError)
		return
	}

	s.writeResponse(w, &response{
		Subscriptions: transformSubscriptions(subs),
		Pagination:    transformPagination(subsPag),
	}, http.StatusOK, nil)
}

// HandleShow receive the request to show a subscription.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	subs, err := s.subscriptionRepository.FindOne(
		r.Context(), s.getResourceID(r), s.getSubscriptionID(r),
	)
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during subscription search", status)
		return
	}

	s.writeResponse(w, transformSubscription(subs), http.StatusOK, nil)
}

// HandleCreate receive the request to create a subscription.
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &subscriptionCreate{}
	)

	if err := d.Decode(content); err != nil {
		s.writeError(w, err, "error during body parse", http.StatusBadRequest)
		return
	}

	if err := content.valid(); err != nil {
		s.writeError(w, err, "invalid body content", http.StatusBadRequest)
		return
	}

	result, err := content.toFlareSubscription()
	if err != nil {
		s.writeError(w, err, "invalid subscription", http.StatusBadRequest)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during resource search", status)
		return
	}
	result.Resource.ID = resource.ID

	if err := s.subscriptionRepository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.AlreadyExists() {
			status = http.StatusConflict
		}

		s.writeError(w, err, "error during subscription create", status)
		return
	}

	header := make(http.Header)
	header.Set("Location", s.getSubscriptionURI(result.Resource.ID, result.ID))
	resp := &response{Subscription: transformSubscription(result)}
	s.writeResponse(w, resp, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a subscription.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	err := s.subscriptionRepository.Delete(r.Context(), s.getResourceID(r), s.getSubscriptionID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during subscription delete", status)
		return
	}

	s.writeResponse(w, nil, http.StatusNoContent, nil)
}

// NewService initialize the service to handle HTTP Requests.
func NewService(options ...func(*Service)) (*Service, error) {
	service := &Service{}

	for _, option := range options {
		option(service)
	}

	if service.subscriptionRepository == nil {
		return nil, errors.New("subscriptionRepository not found")
	}

	if service.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if service.getResourceID == nil {
		return nil, errors.New("getResourceID not found")
	}

	if service.getSubscriptionID == nil {
		return nil, errors.New("getSubscriptionID not found")
	}

	if service.getSubscriptionURI == nil {
		return nil, errors.New("getSubscriptionURI not found")
	}

	if service.parsePagination == nil {
		return nil, errors.New("parsePagination not found")
	}

	if service.writeResponse == nil {
		return nil, errors.New("writeResponse not found")
	}

	return service, nil
}

// ServiceResourceRepository set the repository to access the resources.
func ServiceResourceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.resourceRepository = repo }
}

// ServiceSubscriptionRepository set the repository to access the subscriptions.
func ServiceSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Service) {
	return func(s *Service) { s.subscriptionRepository = repo }
}

// ServiceGetResourceID the function to fetch the resourceId from the URL.
func ServiceGetResourceID(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getResourceID = fn }
}

// ServiceGetSubscriptionID the function to fetch the subscriptionId from the URL.
func ServiceGetSubscriptionID(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getSubscriptionID = fn }
}

// ServiceGetSubscriptionURI set the function to generate the URI or a given subscription.
func ServiceGetSubscriptionURI(fn func(string, string) string) func(*Service) {
	return func(s *Service) { s.getSubscriptionURI = fn }
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
