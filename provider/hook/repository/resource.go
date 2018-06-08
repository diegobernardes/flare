// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"net/url"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

type hook interface {
	Delete(context.Context, string) error
}

// Resource implements the hook for resource repository.
type Resource struct {
	Repository flare.ResourceRepositorier
	Hook       hook
}

// Init check if everything needed is set.
func (r *Resource) Init() error {
	if r.Repository == nil {
		return errors.New("missing repository")
	}

	if r.Hook == nil {
		return errors.New("missing hook")
	}
	return nil
}

// Find is a proxy to the repository find.
func (r *Resource) Find(
	ctx context.Context, pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	return r.Repository.Find(ctx, pagination)
}

// FindByID is a proxy to the repository find by id.
func (r *Resource) FindByID(ctx context.Context, id string) (*flare.Resource, error) {
	return r.Repository.FindByID(ctx, id)
}

// FindByURI is a proxy to the repository find by uri.
func (r *Resource) FindByURI(ctx context.Context, uri url.URL) (*flare.Resource, error) {
	return r.Repository.FindByURI(ctx, uri)
}

// Partitions is a proxy to the repository partitions.
func (r *Resource) Partitions(ctx context.Context, id string) (partitions []string, err error) {
	return r.Repository.Partitions(ctx, id)
}

// Create is a proxy to the repository create.
func (r *Resource) Create(ctx context.Context, resource *flare.Resource) error {
	return r.Repository.Create(ctx, resource)
}

// Delete is a proxy to the repository delete.
func (r *Resource) Delete(ctx context.Context, id string) error {
	if err := r.Repository.Delete(ctx, id); err != nil {
		return err
	}

	if err := r.Hook.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "error during resource hook delete")
	}
	return nil
}
