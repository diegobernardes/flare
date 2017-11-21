// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Document represents the documents from a resource.
type Document struct {
	ID        string
	Revision  int64
	Resource  Resource
	Content   map[string]interface{}
	UpdatedAt time.Time
}

// Valid indicates if the current document is valid.
func (doc *Document) Valid() error {
	if doc.ID == "" {
		return errors.New("missing ID")
	}

	if err := doc.Resource.Change.Valid(); err != nil {
		return errors.Wrap(err, "invalid Resource.Change")
	}
	return nil
}

// Newer indicates if the current document is newer then the one passed as parameter.
func (doc *Document) Newer(reference *Document) bool {
	if reference == nil {
		return true
	}
	return doc.Revision > reference.Revision
}

// DocumentRepositorier used to interact with document data storage.
type DocumentRepositorier interface {
	FindOne(ctx context.Context, id string) (*Document, error)
	FindOneWithRevision(ctx context.Context, id string, revision int64) (*Document, error)
	Update(context.Context, *Document) error
	Delete(ctx context.Context, id string) error
}

// DocumentRepositoryError implements all the errrors the repository can return.
type DocumentRepositoryError interface {
	NotFound() bool
}
