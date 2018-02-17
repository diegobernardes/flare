// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/wildcard"
)

type pagination flare.Pagination

func (p *pagination) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}{
		Limit:  p.Limit,
		Total:  p.Total,
		Offset: p.Offset,
	})
}

type resource flare.Resource

func (r *resource) MarshalJSON() ([]byte, error) {
	change := map[string]string{
		"field": r.Change.Field,
	}

	if r.Change.Format != "" {
		change["format"] = r.Change.Format
	}

	endpoint, err := url.QueryUnescape(r.Endpoint.String())
	if err != nil {
		return nil, errors.Wrap(err, "error during endpoint unescape")
	}

	return json.Marshal(&struct {
		Id        string            `json:"id"`
		Endpoint  string            `json:"endpoint"`
		Change    map[string]string `json:"change"`
		CreatedAt string            `json:"createdAt"`
	}{
		Id:        r.ID,
		Endpoint:  endpoint,
		Change:    change,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
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

func (r *resourceCreate) valid() error {
	if err := r.validEndpoint(); err != nil {
		return errors.Wrap(err, "invalid endpoint")
	}

	if r.Change.Field == "" {
		return errors.New("missing change field")
	}
	return nil
}

func (r *resourceCreate) validEndpoint() error {
	if r.endpoint.Path == "" {
		return errors.New("missing path")
	}

	if r.endpoint.RawQuery != "" {
		return fmt.Errorf("should not have query string '%s'", r.endpoint.RawQuery)
	}

	if r.endpoint.Fragment != "" {
		return fmt.Errorf("should not have fragment '%s'", r.endpoint.Fragment)
	}

	switch r.endpoint.Scheme {
	case "http", "https":
	case "":
		return errors.New("missing scheme")
	default:
		return errors.New("unknown scheme")
	}

	if !wildcard.Present(r.endpoint.Path) {
		return errors.New("missing wildcard")
	}

	if err := wildcard.ValidURL(r.endpoint.Path); err != nil {
		return errors.Wrap(err, "can't have duplicated wildcards")
	}

	return nil
}

func (r *resourceCreate) toFlareResource() *flare.Resource {
	return &flare.Resource{
		ID:       uuid.NewV4().String(),
		Endpoint: r.endpoint,
		Change: flare.ResourceChange{
			Field:  r.Change.Field,
			Format: r.Change.Format,
		},
	}
}

func (r *resourceCreate) normalize() {
	if r.RawEndpoint == "" {
		return
	}

	r.RawEndpoint = strings.TrimSpace(r.RawEndpoint)
	r.RawEndpoint = wildcard.Normalize(r.RawEndpoint)
	if r.RawEndpoint[len(r.RawEndpoint)-1] == '/' {
		r.RawEndpoint = r.RawEndpoint[:len(r.RawEndpoint)-1]
	}
}

func (r *resourceCreate) unescape() error {
	endpoint, err := url.QueryUnescape(r.RawEndpoint)
	if err != nil {
		return errors.Wrap(err, "error during path unescape")
	}
	r.RawEndpoint = endpoint
	return nil
}

func (r *resourceCreate) init() error {
	endpoint, err := url.Parse(r.RawEndpoint)
	if err != nil {
		return errors.Wrap(err, "error during endpoint parse")
	}
	r.endpoint = *endpoint
	return nil
}

type response struct {
	Pagination *pagination
	Resources  []resource
	Resource   *resource
}

func (r *response) MarshalJSON() ([]byte, error) {
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

func transformResources(r []flare.Resource) []resource {
	result := make([]resource, len(r))
	for i := 0; i < len(r); i++ {
		result[i] = (resource)(r[i])
	}
	return result
}

func transformResource(r *flare.Resource) *resource { return (*resource)(r) }

func transformPagination(p *flare.Pagination) *pagination { return (*pagination)(p) }
