// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"net/url"
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

	return json.Marshal(&struct {
		Id        string            `json:"id"`
		Addresses []string          `json:"addresses"`
		Path      string            `json:"path"`
		Change    map[string]string `json:"change"`
		CreatedAt string            `json:"createdAt"`
	}{
		Id:        r.ID,
		Addresses: r.Addresses,
		Path:      r.Path,
		Change:    change,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	})
}

type resourceCreate struct {
	Path      string   `json:"path"`
	Addresses []string `json:"addresses"`
	Change    struct {
		Field  string `json:"field"`
		Format string `json:"format"`
	} `json:"change"`
}

func (r *resourceCreate) valid() error {
	if err := r.validAddresses(); err != nil {
		return err
	}

	if err := r.validPath(); err != nil {
		return err
	}

	if r.Change.Field == "" {
		return errors.New("missing change field")
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

	if !wildcard.Present(r.Path) {
		return errors.New("path is missing a wildcard")
	}

	if err := wildcard.ValidWithoutDuplication(r.Path); err != nil {
		return err
	}

	return nil
}

func (r *resourceCreate) validAddresses() error {
	if len(r.Addresses) == 0 {
		return errors.New("missing addresses")
	}

	for _, address := range r.Addresses {
		if err := r.validAddressUnit(address); err != nil {
			return err
		}
	}

	return nil
}

func (r *resourceCreate) validAddressUnit(addr string) error {
	d, err := url.Parse(addr)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during address parse '%s'", addr))
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

	if wildcard.Present(addr) {
		return errors.New("address cannot have wildcard")
	}

	switch d.Scheme {
	case "http", "https":
		return nil
	case "":
		return errors.Errorf("missing scheme on address '%s'", addr)
	default:
		return errors.Errorf("invalid scheme on address '%s'", addr)
	}
}

func (r *resourceCreate) toFlareResource() *flare.Resource {
	return &flare.Resource{
		ID:        uuid.NewV4().String(),
		Addresses: r.Addresses,
		Path:      r.Path,
		Change: flare.ResourceChange{
			Field:  r.Change.Field,
			Format: r.Change.Format,
		},
	}
}

func (r *resourceCreate) unescape() error {
	for i, rawAddr := range r.Addresses {
		addr, err := url.QueryUnescape(rawAddr)
		if err != nil {
			return errors.Wrap(err, "error during addresses unescape")
		}
		r.Addresses[i] = addr
	}

	path, err := url.QueryUnescape(r.Path)
	if err != nil {
		return errors.Wrap(err, "error during path unescape")
	}
	r.Path = path

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
