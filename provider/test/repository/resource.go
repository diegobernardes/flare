// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Resource implements flare.ResourceRepositorier.
type Resource struct {
	err          error
	findByURIErr error
	base         flare.ResourceRepositorier
	date         time.Time
	createID     string
	partitions   []string
}

// Find mock flare.ResourceRepositorier.FindAll.
func (r *Resource) Find(ctx context.Context, pag *flare.Pagination) (
	[]flare.Resource, *flare.Pagination, error,
) {
	if r.err != nil {
		return nil, nil, r.err
	}

	res, resPag, err := r.base.Find(ctx, pag)
	if err != nil {
		return nil, nil, err
	}

	for i := range res {
		res[i].CreatedAt = r.date
	}

	return res, resPag, nil
}

// FindByID mock flare.ResourceRepositorier.FindOne.
func (r *Resource) FindByID(ctx context.Context, id string) (*flare.Resource, error) {
	if r.err != nil {
		return nil, r.err
	}

	res, err := r.base.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = r.date

	return res, nil
}

// FindByURI mock flare.ResourceRepositorier.FindByURI.
func (r *Resource) FindByURI(ctx context.Context, uri string) (*flare.Resource, error) {
	if r.findByURIErr != nil {
		return nil, r.findByURIErr
	} else if r.err != nil {
		return nil, r.err
	}
	return r.base.FindByURI(ctx, uri)
}

// Create mock flare.ResourceRepositorier.Create.
func (r *Resource) Create(ctx context.Context, resource *flare.Resource) error {
	if r.err != nil {
		return r.err
	}

	err := r.base.Create(ctx, resource)
	resource.CreatedAt = r.date
	resource.ID = r.createID
	return err
}

// Delete mock flare.ResourceRepositorier.Delete.
func (r *Resource) Delete(ctx context.Context, id string) error {
	if r.err != nil {
		return r.err
	}
	return r.base.Delete(ctx, id)
}

// Partitions return the list of partitions of a resource.
func (r *Resource) Partitions(ctx context.Context, id string) (partitions []string, err error) {
	if r.err != nil {
		return nil, r.err
	}

	if len(r.partitions) > 0 {
		return r.partitions, nil
	}

	return r.base.Partitions(ctx, id)
}

func newResource(options ...func(*Resource)) *Resource {
	r := &Resource{}

	for _, option := range options {
		option(r)
	}

	return r
}

// ResourceRepository set the resource repository.
func ResourceRepository(repository flare.ResourceRepositorier) func(*Resource) {
	return func(s *Resource) { s.base = repository }
}

// ResourceCreateID set id used during resource create.
func ResourceCreateID(id string) func(*Resource) {
	return func(r *Resource) { r.createID = id }
}

// ResourceError set the error to be returned during calls.
func ResourceError(err error) func(*Resource) {
	return func(r *Resource) { r.err = err }
}

// ResourceFindByURIError set the error to be returned during findByURI calls.
func ResourceFindByURIError(err error) func(*Resource) {
	return func(r *Resource) { r.findByURIErr = err }
}

// ResourceDate set the date to be used at time fields.
func ResourceDate(date time.Time) func(*Resource) {
	return func(r *Resource) { r.date = date }
}

// ResourceLoadSliceByteResource load a list of encoded resources in a []byte json layout into
// repository.
func ResourceLoadSliceByteResource(content []byte) func(*Resource) {
	return func(r *Resource) {
		resources := make([]struct {
			Id        string    `json:"id"`
			Addresses []string  `json:"addresses"`
			CreatedAt time.Time `json:"createdAt"`
			Path      string    `json:"path"`
			Change    struct {
				Field  string `json:"field"`
				Format string `json:"format"`
			}
		}, 0)
		if err := json.Unmarshal(content, &resources); err != nil {
			panic(errors.Wrap(err,
				fmt.Sprintf("error during unmarshal of '%s' into '%v'", string(content), resources),
			))
		}

		for _, rawResource := range resources {
			err := r.Create(context.Background(), &flare.Resource{
				ID:        rawResource.Id,
				Addresses: rawResource.Addresses,
				Path:      rawResource.Path,
				CreatedAt: rawResource.CreatedAt,
				Change: flare.ResourceChange{
					Format: rawResource.Change.Format,
					Field:  rawResource.Change.Field,
				},
			})
			if err != nil {
				panic(errors.Wrap(err, "error during flare.Resource persistence"))
			}
		}
	}
}

// ResourcePartitions set a list of partitions to be returned by partitions search.
func ResourcePartitions(partitions []string) func(*Resource) {
	return func(r *Resource) {
		r.partitions = partitions
	}
}
