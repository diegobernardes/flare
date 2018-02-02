// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"time"
)

// Document represents the documents from a resource.
type Document struct {
	ID        string
	Revision  int64
	Resource  Resource
	Content   map[string]interface{}
	UpdatedAt time.Time
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
	FindByID(ctx context.Context, id string) (*Document, error)
	Update(context.Context, *Document) error
	Delete(ctx context.Context, id string) error
}

// DocumentRepositoryError implements all the errrors the repository can return.
type DocumentRepositoryError interface {
	error
	NotFound() bool
}
