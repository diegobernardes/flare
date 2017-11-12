// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/memory"
)

// Document implements flare.DocumentRepositorier.
type Document struct {
	base       flare.DocumentRepositorier
	err        error
	deleteErr  error
	findOneErr error
	updateErr  error
	date       time.Time
}

// FindOne mock flare.DocumentRepositorier.FindOne.
func (d *Document) FindOne(ctx context.Context, id string) (*flare.Document, error) {
	if d.findOneErr != nil {
		return nil, d.findOneErr
	} else if d.err != nil {
		return nil, d.err
	}
	document, err := d.base.FindOne(ctx, id)
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
func (d *Document) Delete(ctx context.Context, id string) error {
	if d.deleteErr != nil {
		return d.deleteErr
	} else if d.err != nil {
		return d.err
	}
	return d.base.Delete(ctx, id)
}

// NewDocument return a flare.ResourceRepositorier mock.
func NewDocument(options ...func(*Document)) *Document {
	d := &Document{base: memory.NewDocument()}

	for _, option := range options {
		option(d)
	}

	return d
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
			Id               string      `json:"id"`
			ChangeFieldValue interface{} `json:"changeFieldValue"`
			Resource         struct {
				Id string `json:"id"`
			} `json:"resource"`
		}, 0)
		if err := json.Unmarshal(content, &documents); err != nil {
			panic(errors.Wrap(err,
				fmt.Sprintf("error during unmarshal of '%s' into '%v'", string(content), documents),
			))
		}

		for _, rawDocument := range documents {
			err := d.Update(context.Background(), &flare.Document{
				Id:               rawDocument.Id,
				ChangeFieldValue: rawDocument.ChangeFieldValue,
				Resource: flare.Resource{
					ID: rawDocument.Resource.Id,
				},
			})
			if err != nil {
				panic(errors.Wrap(err, "error during flare.Resource persistence"))
			}
		}
	}
}
