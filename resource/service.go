package resource

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
)

// Service implements the HTTP handler to manage resources.
type Service struct {
	logger          log.Logger
	repository      flare.ResourceRepositorier
	getResourceId   func(*http.Request) string
	getResourceURI  func(string) string
	writeResponse   func(http.ResponseWriter, interface{}, int, http.Header)
	parsePagination func(r *http.Request) (*flare.Pagination, error)
	defaultLimit    int
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

	resources, paginationResponse, err := s.repository.FindAll(r.Context(), pagination)
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
		Pagination: transformPagination(paginationResponse),
		Resources:  transformResources(resources),
	}, http.StatusOK, nil)
}

// HandleShow receive the request to show a given resource.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	re, err := s.repository.FindOne(r.Context(), s.getResourceId(r))
	if err != nil {
		var status int
		if err, ok := err.(flare.ResourceRepositoryError); ok && err.NotFound() {
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
	if re == nil && err == nil {
		s.writeResponse(w, nil, http.StatusNotFound, nil)
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

	result := content.toFlareResource()
	if err := s.repository.Create(r.Context(), result); err != nil {
		status := http.StatusInternalServerError
		if err, ok := err.(flare.ResourceRepositoryError); ok {
			if err.PathConflict() || err.AlreadyExists() {
				status = http.StatusConflict
			}
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
	header.Set("Location", s.getResourceURI(result.Id))
	s.writeResponse(w, &response{Resource: transformResource(result)}, http.StatusCreated, header)
}

// HandleDelete receive the request to delete a resource.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.repository.Delete(r.Context(), s.getResourceId(r)); err != nil {
		status := http.StatusInternalServerError
		if err, ok := err.(flare.ResourceRepositoryError); ok && err.NotFound() {
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

	if service.repository == nil {
		return nil, errors.New("repository not found")
	}

	if service.getResourceId == nil {
		return nil, errors.New("getResourceId not found")
	}

	if service.getResourceURI == nil {
		return nil, errors.New("getResourceURI not found")
	}

	service.parsePagination = infraHTTP.ParsePagination(service.defaultLimit)
	service.writeResponse = infraHTTP.WriteResponse(service.logger)
	return service, nil
}

// ServiceRepository set the repository to access the resources.
func ServiceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.repository = repo }
}

// ServiceGetResourceID the function to fetch the resourceId from the URL.
func ServiceGetResourceID(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getResourceId = fn }
}

// ServiceDefaultLimit set the default value of limit.
func ServiceDefaultLimit(limit int) func(*Service) {
	return func(s *Service) { s.defaultLimit = limit }
}

// ServiceLogger set the logger.
func ServiceLogger(logger log.Logger) func(*Service) {
	return func(s *Service) { s.logger = logger }
}

// ServiceGetResourceURI set the function to generate the URI or a given resource.
func ServiceGetResourceURI(fn func(string) string) func(*Service) {
	return func(s *Service) { s.getResourceURI = fn }
}
