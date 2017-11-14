// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package document

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diegobernardes/flare"
)

type pusher interface {
	push(ctx context.Context, id, action string, body []byte) error
}

type document flare.Document

func (d *document) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id               string      `json:"id"`
		ChangeFieldValue interface{} `json:"changeFieldValue"`
		UpdatedAt        string      `json:"updatedAt"`
	}{
		Id:               d.Id,
		ChangeFieldValue: d.ChangeFieldValue,
		UpdatedAt:        d.UpdatedAt.Format(time.RFC3339),
	})
}

func transformDocument(d *flare.Document) *document {
	return (*document)(d)
}
