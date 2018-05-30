// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Document implements the data layer for the document service.
type Document struct {
	mutex     sync.RWMutex
	documents map[url.URL]flare.Document
}

// FindByID return the document that match the id.
func (d *Document) FindByID(ctx context.Context, id url.URL) (*flare.Document, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	document, ok := d.documents[id]
	if !ok {
		idString, err := url.QueryUnescape(id.String())
		if err != nil {
			return nil, errors.Wrap(err, "error during transform and escape id to string")
		}

		return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", idString), notFound: true}
	}
	return &document, nil
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
func (d *Document) Delete(ctx context.Context, id url.URL) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	delete(d.documents, id)
	return nil
}

// Delete all the documents from a given resource.
func (d *Document) DeleteByResourceID(ctx context.Context, resourceID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for id, document := range d.documents {
		if document.Resource.ID == resourceID {
			delete(d.documents, id)
		}
	}

	return nil
}

func (d *Document) init() { d.documents = make(map[url.URL]flare.Document) }
