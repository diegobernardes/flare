package document

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/log"

	"github.com/diegobernardes/flare"
	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/subscription"
)

// Service implements the HTTP handler to manage documents.
type Service struct {
	logger                 log.Logger
	documentRepository     flare.DocumentRepositorier
	resourceRepository     flare.ResourceRepositorier
	subscriptionRepository flare.SubscriptionRepositorier
	subscriptionTrigger    flare.SubscriptionTrigger
	getDocumentId          func(*http.Request) string
	getDocumentURI         func(id string) string
	writeResponse          func(http.ResponseWriter, interface{}, int, http.Header)
}

// HandleShow receive the request to show a given document.
func (s *Service) HandleShow(w http.ResponseWriter, r *http.Request) {
	d, err := s.documentRepository.FindOne(r.Context(), s.getDocumentId(r))
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
	if d == nil && err == nil {
		s.writeResponse(w, nil, http.StatusNotFound, nil)
		return
	}

	s.writeResponse(w, transformDocument(d), http.StatusOK, nil)
}

// HandleUpdate process the request to update a document.
func (s *Service) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	document, err := s.parseHandleUpdateDocument(w, r)
	if err != nil {
		return
	}

	referenceDocument, err := s.documentRepository.FindOne(r.Context(), document.Id)
	if err != nil {
		if _, ok := err.(flare.DocumentRepositoryError); !ok {
			s.writeResponse(w, &response{
				Error: &responseError{
					Status: http.StatusInternalServerError,
					Title:  "error during document search",
					Detail: err.Error(),
				},
			}, http.StatusInternalServerError, nil)
			return
		}
	}

	hasSubscriptions, err := s.subscriptionRepository.HasSubscription(
		r.Context(), document.Resource.Id,
	)
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during check if the document resource has subscriptions",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return
	}
	if !hasSubscriptions {
		s.writeResponse(w, &response{Document: transformDocument(document)}, http.StatusOK, nil)
		return
	}

	var (
		status int
		header http.Header
	)

	if referenceDocument == nil {
		status = http.StatusCreated
		header = make(http.Header)
		header.Set("Location", s.getDocumentURI(document.Id))
	} else {
		status = http.StatusOK
		newer, err := document.Newer(referenceDocument)
		if err != nil {
			s.writeResponse(w, &response{
				Error: &responseError{
					Status: http.StatusBadRequest,
					Title:  "error during comparing the document with the latest one on datastorage",
					Detail: err.Error(),
				},
			}, http.StatusBadRequest, nil)
			return
		}
		if !newer {
			s.writeResponse(w, &response{Document: transformDocument(referenceDocument)}, status, nil)
			return
		}
	}

	if err := s.updateAndTriggerDocumentChange(w, r, document, status); err != nil {
		return
	}
	s.writeResponse(w, &response{Document: transformDocument(document)}, status, header)
}

func (s *Service) parseHandleUpdateDocument(
	w http.ResponseWriter, r *http.Request,
) (*flare.Document, error) {
	d := json.NewDecoder(r.Body)
	content := make(map[string]interface{})
	if err := d.Decode(&content); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "invalid body content",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return nil, err
	}

	documentId := s.getDocumentId(r)
	resource, err := s.resourceRepository.FindByURI(r.Context(), documentId)
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during resource search",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return nil, err
	}

	document := &flare.Document{
		Id:               documentId,
		ChangeFieldValue: content[resource.Change.Field],
		Resource:         *resource,
	}
	if err = document.Valid(); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusBadRequest,
				Title:  "document is not valid",
				Detail: err.Error(),
			},
		}, http.StatusBadRequest, nil)
		return nil, err
	}

	return document, nil
}

func (s *Service) updateAndTriggerDocumentChange(
	w http.ResponseWriter, r *http.Request, document *flare.Document, status int,
) error {
	if err := s.documentRepository.Update(r.Context(), document); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during document persistence",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return err
	}

	var action string
	switch status {
	case http.StatusCreated:
		action = subscription.TriggerActionCreate
	case http.StatusOK:
		action = subscription.TriggerActionUpdate
	}

	if err := s.subscriptionTrigger.Process(r.Context(), action, document); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during document change trigger",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return err
	}
	return nil
}

// HandleDelete receive the request to delete a document.
func (s *Service) HandleDelete(w http.ResponseWriter, r *http.Request) {
	document, err := s.documentRepository.FindOne(r.Context(), s.getDocumentId(r))
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during the check if the document exists",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return
	}
	if document == nil {
		s.writeResponse(w, nil, http.StatusNotFound, nil)
		return
	}

	if err = s.documentRepository.Delete(r.Context(), document.Id); err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during delete",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return
	}

	err = s.subscriptionTrigger.Process(r.Context(), subscription.TriggerActionDelete, document)
	if err != nil {
		s.writeResponse(w, &response{
			Error: &responseError{
				Status: http.StatusInternalServerError,
				Title:  "error during document change trigger",
				Detail: err.Error(),
			},
		}, http.StatusInternalServerError, nil)
		return
	}

	s.writeResponse(w, nil, http.StatusNoContent, nil)
}

// NewService initialize the service to handle HTTP requests.
func NewService(options ...func(*Service)) (*Service, error) {
	s := &Service{}

	for _, option := range options {
		option(s)
	}

	if s.logger == nil {
		return nil, errors.New("logger not found")
	}

	if s.subscriptionTrigger == nil {
		return nil, errors.New("subscriptionTrigger not found")
	}

	if s.documentRepository == nil {
		return nil, errors.New("documentRepository not found")
	}

	if s.resourceRepository == nil {
		return nil, errors.New("resourceRepository not found")
	}

	if s.subscriptionRepository == nil {
		return nil, errors.New("subscriptionRepository not found")
	}

	if s.getDocumentId == nil {
		return nil, errors.New("getDocumentId not found")
	}

	if s.getDocumentURI == nil {
		return nil, errors.New("getDocumentURI not found")
	}

	s.writeResponse = infraHTTP.WriteResponse(s.logger)
	return s, nil
}

// DocumentDocumentRepository set the repository to access the documents.
func DocumentDocumentRepository(repo flare.DocumentRepositorier) func(*Service) {
	return func(s *Service) { s.documentRepository = repo }
}

// DocumentResourceRepository set the repository to access the resources.
func DocumentResourceRepository(repo flare.ResourceRepositorier) func(*Service) {
	return func(s *Service) { s.resourceRepository = repo }
}

// DocumentSubscriptionRepository set the repository to access the subscriptions.
func DocumentSubscriptionRepository(repo flare.SubscriptionRepositorier) func(*Service) {
	return func(s *Service) { s.subscriptionRepository = repo }
}

// DocumentLogger set the logger.
func DocumentLogger(logger log.Logger) func(*Service) {
	return func(s *Service) { s.logger = logger }
}

// DocumentGetDocumentId set the function to get the document id..
func DocumentGetDocumentId(fn func(*http.Request) string) func(*Service) {
	return func(s *Service) { s.getDocumentId = fn }
}

// DocumentGetDocumentURI set the function to generate the URI of a given Document.
func DocumentGetDocumentURI(fn func(string) string) func(*Service) {
	return func(s *Service) { s.getDocumentURI = fn }
}

// DocumentSubscriptionTrigger set the subscription trigger processor.
func DocumentSubscriptionTrigger(trigger flare.SubscriptionTrigger) func(*Service) {
	return func(s *Service) { s.subscriptionTrigger = trigger }
}
