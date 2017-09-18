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
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "error during pagination parse",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return
	}

	if err = pagination.Valid(); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "invalid pagination",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return
	}

	resource, err := s.resourceRepository.FindOne(r.Context(), s.getResourceId(r))
	if err != nil {
		var status int
		if err, ok := err.(flare.SubscriptionRepositoryError); ok && err.NotFound() {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}

		s.writeResponse(w, &response{
			Error: &responseError{
				Status: status,
				Title:  "error during search",
				Detail: err.Error(),
			},
		}, status, nil)
		return
	}

	subscriptions, paginationResponse, err := s.subscriptionRepository.FindAll(
		r.Context(), pagination, resource.Id,
	)
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during search",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
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
		var status int
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}

		s.writeResponse(w, &response{
			Error: &responseError{
				Status: status,
				Title:  "error during search",
				Detail: err.Error(),
			},
		}, status, nil)
		return
	}
	if subs == nil && err == nil {
		s.writeResponse(w, nil, http.StatusNotFound, nil)
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
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "error during content parse",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return
	}

	if err := content.valid(); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "invalid content",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return
	}

	result, err := content.toFlareSubscription()
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "invalid content",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return
	}
	result.Resource.Id = s.getResourceId(r)
	if err := s.subscriptionRepository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.AlreadyExists() {
			status = http.StatusConflict
		}

		s.writeResponse(w, &response{
			Error: &responseError{
				Status: status,
				Title:  "error during save",
				Detail: err.Error(),
			},
		}, status, nil)
		return
	}

	header := make(http.Header)
	header.Set("Location", s.getSubscriptionURI(result.Resource.Id, result.Id))
	s.writeResponse(w, &response{
		Subscription: transformSubscription(result),
	}, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a resource.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	err := s.subscriptionRepository.Delete(r.Context(), s.getResourceId(r), s.getSubscriptionId(r))
	if err != nil {
		status := http.StatusInternalServerError
		if errRepo, ok := err.(flare.SubscriptionRepositoryError); ok && errRepo.NotFound() {
			status = http.StatusNotFound
		}

		s.writeResponse(w, &response{
			Error: &responseError{
				Status: status,
				Title:  "error during delete",
				Detail: err.Error(),
			},
		}, status, nil)
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
