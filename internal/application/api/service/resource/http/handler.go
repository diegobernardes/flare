package http

import (
	"encoding/json"
	"errors"
	coreHTTP "net/http"

	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
)

type Handler struct {
	ParsePagination func(*coreHTTP.Request) (*infraHTTP.Pagination, error)
	Writer          *infraHTTP.Writer
	Service         service
	ExtractID       func(req *coreHTTP.Request) string
	GenURI          func(id string) string
}

func (h Handler) Init() error {
	if h.ParsePagination == nil {
		return errors.New("missing parse pagination")
	}

	if h.Writer == nil {
		return errors.New("missing writer")
	}

	if h.Service == nil {
		return errors.New("missing service")
	}

	if h.ExtractID == nil {
		return errors.New("missing extract id")
	}

	if h.GenURI == nil {
		return errors.New("missing gen uri")
	}

	return nil
}

func (h Handler) Index(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
	if err := infraHTTP.CheckPermitedQS(r.URL.Query(), []string{"limit", "offset"}); err != nil {
		h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
		return
	}

	pag, err := h.ParsePagination(r)
	if err != nil {
		h.Writer.Error(w, "invalid pagination", err, coreHTTP.StatusBadRequest)
		return
	}

	re, rep, err := h.Service.Find(r.Context(), *pag)
	if err != nil {
		rerr := err.(serviceError)
		if rerr.Client() {
			h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
			return
		}

		h.Writer.Error(w, "error during find", err, coreHTTP.StatusInternalServerError)
		return
	}

	h.Writer.Response(
		w, &response{Resources: transformResources(re), Pagination: rep}, coreHTTP.StatusOK, nil,
	)
}

func (h Handler) Show(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
	if err := infraHTTP.CheckPermitedQS(r.URL.Query(), []string{}); err != nil {
		h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
		return
	}

	re, err := h.Service.FindByID(r.Context(), h.ExtractID(r))
	if err != nil {
		h.Writer.Error(w, "error during find", err, coreHTTP.StatusInternalServerError)
		return
	}
	if re == nil {
		h.Writer.Response(w, nil, coreHTTP.StatusNotFound, nil)
		return
	}

	h.Writer.Response(w, transformResource(re), coreHTTP.StatusOK, nil)
}

func (h Handler) Create(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
	if err := infraHTTP.CheckPermitedQS(r.URL.Query(), []string{}); err != nil {
		h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	rawResource := &resourceCreate{}
	if err := decoder.Decode(rawResource); err != nil {
		h.Writer.Error(w, "error during body parse", err, coreHTTP.StatusBadRequest)
		return
	}

	if err := rawResource.init(); err != nil {
		h.Writer.Error(w, "error during resource initialization", err, coreHTTP.StatusBadRequest)
		return
	}

	resource := rawResource.toResource()
	id, err := h.Service.Create(r.Context(), resource)
	if err != nil {
		rerr := err.(serviceError)

		status := coreHTTP.StatusInternalServerError
		if rerr.AlreadyExists() {
			status = coreHTTP.StatusConflict
		}

		h.Writer.Error(w, "error during resource create", err, status)
		return
	}

	header := make(coreHTTP.Header)
	header.Set("Location", h.GenURI(id))

	resource.ID = id
	h.Writer.Response(
		w, &response{Resource: transformResource(&resource)}, coreHTTP.StatusCreated, header,
	)
}

// Delete receive the request to delete a resource.
func (h Handler) Delete(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
	if err := infraHTTP.CheckPermitedQS(r.URL.Query(), []string{}); err != nil {
		h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
		return
	}

	if err := h.Service.Delete(r.Context(), h.ExtractID(r)); err != nil {
		rerr := err.(serviceError)

		status := coreHTTP.StatusInternalServerError
		if rerr.NotFound() {
			status = coreHTTP.StatusNotFound
		}

		h.Writer.Error(w, "error during resource delete", err, status)
		return
	}

	h.Writer.Response(w, nil, coreHTTP.StatusNoContent, nil)
}
