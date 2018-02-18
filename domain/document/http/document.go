// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	infraURL "github.com/diegobernardes/flare/infra/url"
)

type document flare.Document

func (d *document) MarshalJSON() ([]byte, error) {
	id, err := infraURL.String(d.ID)
	if err != nil {
		return nil, errors.Wrap(err, "error during document.ID transform to string")
	}

	return json.Marshal(&struct {
		ID        string                 `json:"id"`
		UpdatedAt string                 `json:"updatedAt"`
		Content   map[string]interface{} `json:"content"`
	}{
		ID:        id,
		UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
		Content:   d.Content,
	})
}

func (d *document) parseBody(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&d.Content); err != nil {
		return errors.Wrap(err, "error during body unmarshal")
	}
	return nil
}

func (d *document) parseRevision() error {
	switch value := d.Content[d.Resource.Change.Field].(type) {
	case string:
		format := d.Resource.Change.Format
		revision, err := time.Parse(format, value)
		if err != nil {
			return errors.Wrapf(err, "error during parse time using '%s' with format '%s'", value, format)
		}
		d.Revision = revision.UnixNano()
	case float64:
		d.Revision = int64(value)
	default:
		return errors.New("data type not supported")
	}
	return nil
}

func validEndpoint(endpoint *url.URL) error {
	if endpoint.Opaque != "" {
		return fmt.Errorf("should not have opaque content '%s'", endpoint.Opaque)
	}

	if endpoint.User != nil {
		return errors.New("should not have user")
	}

	if endpoint.Host == "" {
		return errors.New("missing host")
	}

	if endpoint.Path == "" {
		return errors.New("missing path")
	}

	if endpoint.RawQuery != "" {
		return fmt.Errorf("should not have query string '%s'", endpoint.RawQuery)
	}

	if endpoint.Fragment != "" {
		return fmt.Errorf("should not have fragment '%s'", endpoint.Fragment)
	}

	switch endpoint.Scheme {
	case "http", "https":
	case "":
		return errors.New("missing scheme")
	default:
		return errors.Errorf("unknown scheme '%s'", endpoint.Scheme)
	}

	return nil
}

func marshal(d *document) *flare.Document { return (*flare.Document)(d) }

func unmarshal(d *flare.Document) *document { return (*document)(d) }
