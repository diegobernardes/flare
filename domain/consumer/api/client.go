package api

import (
	"encoding/json"
	"errors"
	"net/http"

	infraHTTP "github.com/diegobernardes/flare/infra/http"
	"github.com/diegobernardes/flare/infra/pagination"
)

// Client implements the HTTP handler to manage consumers.
type Client struct {
	Repository      ClientRepository
	GetID           func(*http.Request) string
	GetURI          func(string) string
	ParsePagination func(r *http.Request) (*pagination.Pagination, error)
	Writer          *infraHTTP.Writer
}

// Index receive the request to list the consumers.
func (c *Client) Index(w http.ResponseWriter, r *http.Request) {
	pag, err := c.ParsePagination(r)
	if err != nil {
		c.Writer.Error(w, "error during pagination parse", err, http.StatusBadRequest)
		return
	}

	if err = pag.Valid(); err != nil {
		c.Writer.Error(w, "invalid pagination", err, http.StatusBadRequest)
		return
	}

	re, pagination, err := c.Repository.Find(r.Context(), pag)
	if err != nil {
		c.Writer.Error(w, "error during resources search", err, http.StatusInternalServerError)
		return
	}

	c.Writer.Response(w, &response{
		Consumers:  transformConsumers(re),
		Pagination: pagination,
	}, http.StatusOK, nil)
}

func (c *Client) Show(w http.ResponseWriter, r *http.Request) {
	consumer, err := c.Repository.FindByID(r.Context(), c.GetID(r))
	if err != nil {
		c.Writer.Error(w, "error during consumer search", err, http.StatusInternalServerError)
		return
	}

	c.Writer.Response(w, &response{Consumer: transformConsumer(consumer)}, http.StatusOK, nil)
}

func (c *Client) Create(w http.ResponseWriter, r *http.Request) {
	var (
		d       = json.NewDecoder(r.Body)
		content = &consumerCreate{}
	)

	if err := d.Decode(content); err != nil {
		c.Writer.Error(w, "error during body parse", err, http.StatusBadRequest)
		return
	}

	if err := content.init(); err != nil {
		c.Writer.Error(w, "invalid body", err, http.StatusBadRequest)
		return
	}

	base := content.marshal()
	if err := c.Repository.Create(r.Context(), base); err != nil {
		c.Writer.Error(w, "error during consumer create", err, http.StatusBadRequest)
		return
	}

	header := make(http.Header)
	header.Set("Location", c.GetURI(content.ID))
	c.Writer.Response(w, &response{Consumer: transformConsumer(base)}, http.StatusCreated, header)
}

func (c *Client) Update(w http.ResponseWriter, r *http.Request) {
}

// Tem que verificar se tem producer antes de deletar consumer.
func (c *Client) Delete(w http.ResponseWriter, r *http.Request) {
	if err := c.Repository.Delete(r.Context(), c.GetID(r)); err != nil {
		c.Writer.Error(w, "error during consumer delete", err, http.StatusInternalServerError)
		return
	}

	c.Writer.Response(w, nil, http.StatusAccepted, nil)

	// id := h.parseID(w, r)
	// if id == nil {
	// 	return
	// }

	// resource := h.fetchResource(r.Context(), id, w)
	// if resource == nil {
	// 	return
	// }

	// doc := &flare.Document{
	// 	ID:        *id,
	// 	UpdatedAt: time.Now(),
	// 	Resource:  *resource,
	// }
	// action := flare.SubscriptionTriggerDelete

	// if err := h.subscriptionTrigger.Push(r.Context(), doc, action); err != nil {
	// 	h.writer.Error(w, "error during subscription trigger", err, http.StatusInternalServerError)
	// 	return
	// }

	// h.writer.Response(w, nil, http.StatusAccepted, nil)
}

func (c *Client) Init() error {
	if c.Repository == nil {
		return errors.New("missing repository")
	}

	if c.GetID == nil {
		return errors.New("missing getID")
	}

	if c.GetURI == nil {
		return errors.New("missing getURI")
	}

	if c.ParsePagination == nil {
		return errors.New("missing parsePagination")
	}

	if c.Writer == nil {
		return errors.New("missing writer")
	}

	return nil
}
