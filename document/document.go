// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

type document flare.Document

func (d *document) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        string                 `json:"id"`
		UpdatedAt string                 `json:"updatedAt"`
		Content   map[string]interface{} `json:"content"`
	}{
		ID:        d.ID,
		UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
		Content:   d.Content,
	})
}

func parseDocument(
	id string, rawContent []byte, resource *flare.Resource,
) (*flare.Document, error) {
	doc := &flare.Document{
		ID:        id,
		Content:   make(map[string]interface{}),
		Resource:  *resource,
		UpdatedAt: time.Now(),
	}

	if err := json.Unmarshal(rawContent, &doc.Content); err != nil {
		return nil, errors.Wrap(err, "error during content unmarshal")
	}

	switch value := doc.Content[doc.Resource.Change.Field].(type) {
	case string:
		revision, err := time.Parse(resource.Change.Format, value)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error during time parse with value '%s' and format '%s'",
				value,
				resource.Change.Format,
			)
		}
		doc.Revision = revision.UnixNano()
	case float64:
		doc.Revision = int64(value)
	default:
		return nil, errors.New("format not supported")
	}
	return doc, nil
}
