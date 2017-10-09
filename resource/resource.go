// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package resource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
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
		"kind":  r.Change.Kind,
		"field": r.Change.Field,
	}

	if r.Change.DateFormat != "" {
		change["dateFormat"] = r.Change.DateFormat
	}

	return json.Marshal(&struct {
		Id        string            `json:"id"`
		Addresses []string          `json:"addresses"`
		Path      string            `json:"path"`
		Change    map[string]string `json:"change"`
		CreatedAt string            `json:"createdAt"`
	}{
		Id:        r.Id,
		Addresses: r.Addresses,
		Path:      r.Path,
		Change:    change,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	})
}

type resourceCreateChange struct {
	Kind       string `json:"kind"`
	Field      string `json:"field"`
	DateFormat string `json:"dateFormat"`
}

type resourceCreate struct {
	Path      string               `json:"path"`
	Addresses []string             `json:"addresses"`
	Change    resourceCreateChange `json:"change"`
}

func (r *resourceCreate) valid() error {
	if err := r.validAddresses(); err != nil {
		return err
	}

	if err := r.validPath(); err != nil {
		return err
	}

	if r.Change.Field == "" {
		return errors.New("missing change")
	}

	switch r.Change.Kind {
	case flare.ResourceChangeInteger, flare.ResourceChangeString:
	case flare.ResourceChangeDate:
		if r.Change.DateFormat == "" {
			return errors.New("missing change.dateFormat")
		}
	default:
		return errors.New("invalid change.kind")
	}

	return nil
}

func (r *resourceCreate) validPath() error {
	if r.Path == "" {
		return errors.New("missing path")
	}

	if r.Path[0] != '/' {
		return errors.New("path should start with a slash")
	}

	if r.Path[len(r.Path)-1] == '/' {
		return errors.New("path should not end with a slash")
	}

	if err := r.validWildcard(); err != nil {
		return err
	}

	return nil
}

func (r *resourceCreate) validWildcard() error {
	var (
		wildcards   = make(map[string]struct{})
		hasWildcard bool
	)

	for _, value := range strings.Split(r.Path, "/") {
		if value == "" {
			continue
		}

		if value[0] == '{' && value[len(value)-1] == '}' {
			if _, ok := wildcards[value]; ok {
				return fmt.Errorf("wildcard '%s' is present %d times", value, strings.Count(r.Path, value))
			}
			wildcards[value] = struct{}{}
			hasWildcard = true
		}

		if strings.Count(value, "{") > 1 || strings.Count(value, "}") > 1 {
			return errors.New("could not use brackets inside a wildcard or next to another wildcard")
		}
	}

	if !hasWildcard {
		return errors.New("missing wildcard")
	}
	return nil
}

func (r *resourceCreate) validAddresses() error {
	if len(r.Addresses) == 0 {
		return errors.New("missing addresses")
	}

	for _, address := range r.Addresses {
		d, err := url.Parse(address)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error during address parse '%s'", address))
		}

		if d.Path != "" {
			return fmt.Errorf("address is invalid because it has a path '%s'", d.Path)
		}

		if d.RawQuery != "" {
			return fmt.Errorf("address is invalid because it has query string '%s'", d.RawQuery)
		}

		if d.Fragment != "" {
			return fmt.Errorf("address is invalid because it has fragment '%s'", d.Fragment)
		}

		switch d.Scheme {
		case "http", "https":
			continue
		case "":
			return errors.Errorf("missing scheme on address '%s'", address)
		default:
			return errors.Errorf("invalid scheme on address '%s'", address)
		}
	}

	return nil
}

func (r *resourceCreate) toFlareResource() *flare.Resource {
	return &flare.Resource{
		Id:        uuid.NewV4().String(),
		Addresses: r.Addresses,
		Path:      r.Path,
		Change: flare.ResourceChange{
			Kind:       r.Change.Kind,
			Field:      r.Change.Field,
			DateFormat: r.Change.DateFormat,
		},
	}
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

type response struct {
	Pagination *pagination
	Error      *responseError
	Resources  []resource
	Resource   *resource
}

func (r *response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Error != nil {
		result = map[string]*responseError{"error": r.Error}
	} else if r.Resource != nil {
		result = r.Resource
	} else {
		result = map[string]interface{}{
			"pagination": r.Pagination,
			"resources":  r.Resources,
		}
	}

	return json.Marshal(result)
}

type responseError struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

func (s *Service) writeError(w http.ResponseWriter, err error, title string, status int) {
	resp := &response{Error: &responseError{Status: status}}

	if err != nil {
		resp.Error.Detail = err.Error()
	}

	if title != "" {
		resp.Error.Title = title
	}

	s.writeResponse(w, resp, status, nil)
}
