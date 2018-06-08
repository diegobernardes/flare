// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// DocumentErr implements the repository error.
type DocumentErr struct {
	Message string
	NF      bool
}

// NotFound indicate if the document is not found.
func (d *DocumentErr) NotFound() bool { return d.NF }

// Error return the error cause.
func (d *DocumentErr) Error() string { return d.Message }

// Document implements flare.DocumentRepositorier.
type Document struct {
	base       flare.DocumentRepositorier
	err        error
	deleteErr  error
	findOneErr error
	updateErr  error
	date       time.Time
}

// FindByID mock flare.DocumentRepositorier.FindOne.
func (d *Document) FindByID(ctx context.Context, id url.URL) (*flare.Document, error) {
	if d.findOneErr != nil {
		return nil, d.findOneErr
	} else if d.err != nil {
		return nil, d.err
	}
	document, err := d.base.FindByID(ctx, id)
	if err != nil {
		return document, err
	}
	document.UpdatedAt = d.date
	return document, err
}

// Update mock flare.DocumentRepositorier.Update.
func (d *Document) Update(ctx context.Context, document *flare.Document) error {
	if d.updateErr != nil {
		return d.updateErr
	} else if d.err != nil {
		return d.err
	}
	err := d.base.Update(ctx, document)
	document.UpdatedAt = d.date
	return err
}

// Delete mock flare.DocumentRepositorier.Delete.
func (d *Document) Delete(ctx context.Context, id url.URL) error {
	if d.deleteErr != nil {
		return d.deleteErr
	} else if d.err != nil {
		return d.err
	}
	return d.base.Delete(ctx, id)
}

// DeleteByResourceID mock flare.DocumentRepositorier.DeleteByResourceID.
func (d *Document) DeleteByResourceID(ctx context.Context, resourceID string) error {
	if d.deleteErr != nil {
		return d.deleteErr
	} else if d.err != nil {
		return d.err
	}
	return d.base.DeleteByResourceID(ctx, resourceID)
}

func newDocument(options ...func(*Document)) *Document {
	d := &Document{}

	for _, option := range options {
		option(d)
	}

	return d
}

// DocumentRepository set the document repository.
func DocumentRepository(repository flare.DocumentRepositorier) func(*Document) {
	return func(d *Document) { d.base = repository }
}

// DocumentError set the error to be returned during calls.
func DocumentError(err error) func(*Document) {
	return func(d *Document) { d.err = err }
}

// DocumentUpdateError set the error to be returned update calls.
func DocumentUpdateError(err error) func(*Document) {
	return func(d *Document) { d.updateErr = err }
}

// DocumentDeleteError set the error to be returned during delete calls.
func DocumentDeleteError(err error) func(*Document) {
	return func(d *Document) { d.deleteErr = err }
}

// DocumentFindOneError set the error to be returned during findOne calls.
func DocumentFindOneError(err error) func(*Document) {
	return func(d *Document) { d.findOneErr = err }
}

// DocumentDate set the date to be used at time fields.
func DocumentDate(date time.Time) func(*Document) {
	return func(d *Document) { d.date = date }
}

// DocumentLoadSliceByteDocument load a list of encoded documents in a []byte json layout into
// repository.
func DocumentLoadSliceByteDocument(content []byte) func(*Document) {
	return func(d *Document) {
		documents := make([]struct {
			ID       string                 `json:"id"`
			Revision float64                `json:"revision"`
			Content  map[string]interface{} `json:"content"`
			Resource struct {
				ID string `json:"id"`
			} `json:"resource"`
		}, 0)
		if err := json.Unmarshal(content, &documents); err != nil {
			panic(errors.Wrap(err,
				fmt.Sprintf("error during unmarshal of '%s' into '%v'", string(content), documents),
			))
		}

		for _, rawDocument := range documents {
			id, err := url.Parse(rawDocument.ID)
			if err != nil {
				panic(errors.Wrap(err, "error during parse of string to url.URL"))
			}

			err = d.Update(context.Background(), &flare.Document{
				ID:       *id,
				Revision: (int64)(rawDocument.Revision),
				Content:  rawDocument.Content,
				Resource: flare.Resource{
					ID: rawDocument.Resource.ID,
				},
			})
			if err != nil {
				panic(errors.Wrap(err, "error during flare.Resource persistence"))
			}
		}
	}
}
