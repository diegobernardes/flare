// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Resource represents the apis Flare track and the info to detect changes on documents.
type Resource struct {
	ID        string
	Addresses []string
	Path      string
	Change    ResourceChange
	CreatedAt time.Time
}

// ResourceChange holds the information to detect document change.
type ResourceChange struct {
	Field  string
	Format string
}

// Valid indicates if the current resourceChange is valid.
func (rc *ResourceChange) Valid() error {
	if rc.Field == "" {
		return errors.New("missing field")
	}
	return nil
}

// ResourceRepositorier is used to interact with Resource repository.
type ResourceRepositorier interface {
	Find(context.Context, *Pagination) ([]Resource, *Pagination, error)
	FindByID(context.Context, string) (*Resource, error)
	FindByURI(context.Context, string) (*Resource, error)
	Partitions(ctx context.Context, id string) (partitions []string, err error)
	Create(context.Context, *Resource) error
	Delete(context.Context, string) error
}

// ResourceRepositoryError represents all the errors the repository can return.
type ResourceRepositoryError interface {
	error
	AlreadyExists() bool
	NotFound() bool
}
