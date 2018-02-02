// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Document implements the data layer for the document service.
type Document struct {
	mutex     sync.RWMutex
	documents map[string]flare.Document
}

// FindByID return the document that match the id.
func (d *Document) FindByID(ctx context.Context, id string) (*flare.Document, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	document, ok := d.documents[id]
	if !ok {
		return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", id), notFound: true}
	}
	return &document, nil
}

// FindByIDAndRevision return the document that match the id and the revision.
func (d *Document) FindByIDAndRevision(
	context.Context, string, int64,
) (*flare.Document, error) {
	return nil, errors.New("not implemented")
}

// Update a document.
func (d *Document) Update(ctx context.Context, doc *flare.Document) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	doc.UpdatedAt = time.Now()
	d.documents[doc.ID] = *doc
	return nil
}

// Delete a given document.
func (d *Document) Delete(ctx context.Context, id string) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	delete(d.documents, id)
	return nil
}

func (d *Document) init() { d.documents = make(map[string]flare.Document) }
