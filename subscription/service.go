// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Service implements the HTTP handler to manage subscriptions.
type Service struct {
	logger                 log.Logger
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	getResourceId          func(*http.Request) string
	getSubscriptionId      func(*http.Request) string
	getSubscriptionURI     func(string, string) string
	writeResponse          func(http.ResponseWriter, interface{}, int, http.Header)
	parsePagination        func(r *http.Request) (*flare.Pagination, error)
	defaultLimit           int
}

// HandleIndex receive the request to list the resources.
func (s *Service) HandleIndex(w http.ResponseWriter, r *http.Request) {
	pagination, err := s.parsePagination(r)
	if err != nil {
		s.writeError(w, err, "error during pagination parse", http.StatusBadRequest)
		return
	}

	if err = pagination.Valid(); err != nil {
		s.writeError(w, err, "invalid pagination", http.StatusBadRequest)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceId(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.ResourceRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeError(w, err, "error during resource search", status)
		return
	}

	subscriptions, paginationResponse, err := s.subscriptionRepository.FindAll(
		r.Context(), pagination, resource.ID,
	)
	if err != nil {
		s.writeError(w, err, "error during subscription search", http.StatusInternalServerError)
		return
	}

	s.writeResponse(w, &response{
		Pagination:    transformPagination(paginationResponse),
		Subscriptions: transformSubscriptions(subscriptions),
	}, http.StatusOK, nil)
}

// HandleShow receive the request to get a resource.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	subs, err := s.subscriptionRepository.FindOne(
		r.Context(), s.getResourceId(r), s.getSubscriptionId(r),
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

// HandleCreate receive the request to create a resource.
func (s *Service) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &subscriptionCreate{}
	)

	if err := d.Decode(content); err != nil {
		s.writeError(w, err, "error during content parse", http.StatusBadRequest)
		return
	}

	if err := content.valid(); err != nil {
		s.writeError(w, err, "invalid content", http.StatusBadRequest)
		return
	}

	result, err := content.toFlareSubscription()
	if err != nil {
		s.writeError(w, err, "invalid content", http.StatusBadRequest)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceId(r))
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

		s.writeError(w, err, "error during subscription save", status)
		return
	}

	header := make(http.Header)
	header.Set("Location", s.getSubscriptionURI(result.Resource.ID, result.ID))
	resp := &response{Subscription: transformSubscription(result)}
	s.writeResponse(w, resp, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a resource.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	err := s.subscriptionRepository.Delete(r.Context(), s.getResourceId(r), s.getSubscriptionId(r))
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

	if service.defaultLimit < 0 {
		return nil, fmt.Errorf("invalid defaultLimit '%d'", service.defaultLimit)
	} else if service.defaultLimit == 0 {
		service.defaultLimit = 30
	}

	if service.logger == nil {
		return nil, errors.New("logger not found")
	}

	if service.subscriptionRepository == nil {
		return nil, errors.New("subscriptionRepository not found")
	}

	if service.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if service.getResourceId == nil {
		return nil, errors.New("getResourceId not found")
	}

	if service.getSubscriptionId == nil {
		return nil, errors.New("getSubscriptionId not found")
	}

	if service.getSubscriptionURI == nil {
		return nil, errors.New("getSubscriptionURI not found")
	}

	service.parsePagination = infraHTTP.ParsePagination(service.defaultLimit)
	service.writeResponse = infraHTTP.WriteResponse(service.logger)
	return service, nil
}

// ServiceLogger set the logger.
func ServiceLogger(logger log.Logger) func(*Service) {
	return func(s *Service) { s.logger = logger }
}

// ServiceResourceRepository set the repository to access the resources.
func ServiceResourceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.resourceRepository = repo }
}

// ServiceSubscriptionRepository set the repository to access the subscriptions.
func ServiceSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Service) {
	return func(s *Service) { s.subscriptionRepository = repo }
}

// ServiceGetResourceId the function to fetch the resourceId from the URL.
func ServiceGetResourceId(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getResourceId = fn }
}

// ServiceGetSubscriptionId the function to fetch the subscriptionId from the URL.
func ServiceGetSubscriptionId(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getSubscriptionId = fn }
}

// ServiceGetSubscriptionURI set the function to generate the URI or a given subscription.
func ServiceGetSubscriptionURI(fn func(string, string) string) func(*Service) {
	return func(s *Service) { s.getSubscriptionURI = fn }
}

// ServiceDefaultLimit set the default value of limit.
func ServiceDefaultLimit(limit int) func(*Service) {
	return func(s *Service) { s.defaultLimit = limit }
}
