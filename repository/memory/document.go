// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/diegobernardes/flare"
)

// Document implements the data layer for the document service.
type Document struct {
	mutex     sync.RWMutex
	documents map[string]flare.Document
}

// FindOne return the document that match the id.
func (d *Document) FindOne(ctx context.Context, id string) (*flare.Document, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	document, ok := d.documents[id]
	if !ok {
		return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", id), notFound: true}
	}
	return &document, nil
}

// Update a document.
func (d *Document) Update(ctx context.Context, doc *flare.Document) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	doc.UpdatedAt = time.Now()
	d.documents[doc.Id] = *doc
	return nil
}

// Delete a given document.
func (d *Document) Delete(ctx context.Context, id string) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	delete(d.documents, id)
	return nil
}

// NewDocument returns a configured document repository.
func NewDocument() *Document {
	return &Document{documents: make(map[string]flare.Document)}
}
