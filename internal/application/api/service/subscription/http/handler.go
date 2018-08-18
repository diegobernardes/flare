package http

import (
	"encoding/json"
	"errors"
	coreHTTP "net/http"

	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
)

type Handler struct {
	ParsePagination       func(*coreHTTP.Request) (*infraHTTP.Pagination, error)
	Writer                infraHTTP.Writer
	Service               service
	ExtractSubscriptionID func(req *coreHTTP.Request) string
	ExtractResourceID     func(req *coreHTTP.Request) string
	GenURI                func(id string) string
}

func (h Handler) Init() error {
	if h.ParsePagination == nil {
		return errors.New("missing parse pagination")
	}

	if h.Service == nil {
		return errors.New("missing service")
	}

	if h.ExtractResourceID == nil {
		return errors.New("missing extract resource id")
	}

	if h.ExtractSubscriptionID == nil {
		return errors.New("missing extract resource id")
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

	s, sp, err := h.Service.Find(r.Context(), *pag)
	if err != nil {
		rerr := err.(serviceError)
		if rerr.Client() {
			h.Writer.Error(w, "invalid request", err, coreHTTP.StatusBadRequest)
			return
		}

		h.Writer.Error(w, "error during find", err, coreHTTP.StatusInternalServerError)
		return
	}

	h.Writer.Response(w, &response{
		Subscriptions: transformSubscriptions(s), Pagination: sp,
	}, coreHTTP.StatusOK, nil)
}

func (h Handler) Show(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
}

func (h Handler) Create(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	rawSubscription := &subscriptionCreate{}
	if err := decoder.Decode(rawSubscription); err != nil {
		h.Writer.Error(w, "error during body parse", err, coreHTTP.StatusBadRequest)
		return
	}

	if err := rawSubscription.init(); err != nil {
		h.Writer.Error(w, "error during subscription initialization", err, coreHTTP.StatusBadRequest)
		return
	}

	resource, err := h.Service.FindResource(r.Context(), h.ExtractResourceID(r))
	if err != nil {
		panic(err)
	}
	_ = resource

	// resource := rawResource.toResource()
	// id, err := h.Service.Create(r.Context(), resource)
	// if err != nil {
	// 	rerr := err.(serviceError)

	// 	status := coreHTTP.StatusInternalServerError
	// 	if rerr.AlreadyExists() {
	// 		status = coreHTTP.StatusConflict
	// 	}

	// 	h.Writer.Error(w, "error during resource create", err, status)
	// 	return
	// }

	// header := make(coreHTTP.Header)
	// header.Set("Location", h.GenURI(id))

	// resource.ID = id
	// h.Writer.Response(
	// 	w, &response{Resource: transformResource(&resource)}, coreHTTP.StatusCreated, header,
	// )
}

func (h Handler) Delete(w coreHTTP.ResponseWriter, r *coreHTTP.Request) {}
