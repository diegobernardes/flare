package http

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/internal"
	infraHTTP "github.com/diegobernardes/flare/internal/application/api/infra/http"
)

type service interface {
	Find(
		ctx context.Context, pagination infraHTTP.Pagination,
	) ([]internal.Resource, infraHTTP.Pagination, error)
	FindByID(ctx context.Context, resourceID string) (*internal.Resource, error)
	Create(ctx context.Context, resource internal.Resource) (string, error)
	Delete(ctx context.Context, resourceID string) error
}

type serviceError interface {
	error
	Client() bool
	Server() bool
	NotFound() bool
	AlreadyExists() bool
}

type resource internal.Resource

func (r resource) MarshalJSON() ([]byte, error) {
	change := map[string]string{
		"field": r.Change.Field,
	}

	if r.Change.Format != "" {
		change["format"] = r.Change.Format
	}

	// TODO: this is really needed?
	endpoint, err := url.QueryUnescape(r.Endpoint.String())
	if err != nil {
		return nil, errors.Wrap(err, "error during endpoint unescape")
	}

	return json.Marshal(&struct {
		ID       string            `json:"id"`
		Endpoint string            `json:"endpoint"`
		Change   map[string]string `json:"change"`
	}{
		ID:       r.ID,
		Endpoint: endpoint,
		Change:   change,
	})
}

type resourceCreate struct {
	endpoint    url.URL
	RawEndpoint string `json:"endpoint"`
	Change      struct {
		Field  string `json:"field"`
		Format string `json:"format"`
	} `json:"change"`
}

func (r *resourceCreate) init() error {
	endpoint, err := url.Parse(r.RawEndpoint)
	if err != nil {
		return errors.Wrap(err, "error during parse endpoint")
	}
	r.endpoint = *endpoint
	return nil
}

func (r resourceCreate) toResource() internal.Resource {
	return internal.Resource{
		Endpoint: r.endpoint,
		Change: internal.ResourceChange{
			Field:  r.Change.Field,
			Format: r.Change.Format,
		},
	}
}

type response struct {
	Pagination infraHTTP.Pagination
	Resources  []resource
	Resource   *resource
}

func (r response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Resource != nil {
		result = r.Resource
	} else {
		result = map[string]interface{}{
			"pagination": r.Pagination,
			"resources":  r.Resources,
		}
	}

	return json.Marshal(result)
}

func transformResources(r []internal.Resource) []resource {
	result := make([]resource, len(r))
	for i := 0; i < len(r); i++ {
		result[i] = (resource)(r[i])
	}
	return result
}

func transformResource(r *internal.Resource) *resource { return (*resource)(r) }
