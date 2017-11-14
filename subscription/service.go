// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Service implements the HTTP handler to manage subscriptions.
type Service struct {
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	getResourceID          func(*http.Request) string
	getSubscriptionID      func(*http.Request) string
	getSubscriptionURI     func(string, string) string
	writer                 *infraHTTP.Writer
	parsePagination        func(r *http.Request) (*flare.Pagination, error)
}

// HandleIndex receive the request to list the subscriptions.
func (s *Service) HandleIndex(w http.ResponseWriter, r *http.Request) {
	pag, err := s.parsePagination(r)
	if err != nil {
		s.writer.Error(w, "error during pagination parse", err, http.StatusBadRequest)
		return
	}

	if err = pag.Valid(); err != nil {
		s.writer.Error(w, "invalid pagination", err, http.StatusBadRequest)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writer.Error(w, "error during resource search", err, status)
		return
	}

	subs, subsPag, err := s.subscriptionRepository.FindAll(
		r.Context(), pag, resource.ID,
	)
	if err != nil {
		s.writer.Error(w, "error during subscriptions search", err, http.StatusInternalServerError)
		return
	}

	s.writer.Response(w, &response{
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

		s.writer.Error(w, "error during subscription search", err, status)
		return
	}

	s.writer.Response(w, transformSubscription(subs), http.StatusOK, nil)
}

// HandleCreate receive the request to create a subscription.
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &subscriptionCreate{}
	)

	if err := d.Decode(content); err != nil {
		s.writer.Error(w, "error during body parse", err, http.StatusBadRequest)
		return
	}

	if err := content.valid(); err != nil {
		s.writer.Error(w, "invalid body content", err, http.StatusBadRequest)
		return
	}

	result, err := content.toFlareSubscription()
	if err != nil {
		s.writer.Error(w, "invalid subscription", err, http.StatusBadRequest)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writer.Error(w, "error during resource search", err, status)
		return
	}
	result.Resource.ID = resource.ID

	if err := s.subscriptionRepository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.AlreadyExists() {
			status = http.StatusConflict
		}

		s.writer.Error(w, "error during subscription create", err, status)
		return
	}

	header := make(http.Header)
	header.Set("Location", s.getSubscriptionURI(result.Resource.ID, result.ID))
	resp := &response{Subscription: transformSubscription(result)}
	s.writer.Response(w, resp, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a subscription.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	err := s.subscriptionRepository.Delete(r.Context(), s.getResourceID(r), s.getSubscriptionID(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writer.Error(w, "error during subscription delete", err, status)
		return
	}

	s.writer.Response(w, nil, http.StatusNoContent, nil)
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

	if service.writer == nil {
		return nil, errors.New("writer not found")
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

// ServiceWriter set the function that return the content to client.
func ServiceWriter(writer *infraHTTP.Writer) func(*Service) {
	return func(s *Service) {
		s.writer = writer
	}
}
