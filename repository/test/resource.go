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

// Resource implements flare.ResourceRepositorier.
type Resource struct {
	err  error
	base flare.ResourceRepositorier
	date time.Time
}

// FindAll mock flare.ResourceRepositorier.FindAll.
func (r *Resource) FindAll(context.Context, *flare.Pagination) (
	[]flare.Resource, *flare.Pagination, error,
) {
	return nil, nil, nil
}

// FindOne mock flare.ResourceRepositorier.FindOne.
func (r *Resource) FindOne(ctx context.Context, id string) (*flare.Resource, error) {
	if r.err != nil {
		return nil, r.err
	}

	res, err := r.base.FindOne(ctx, id)
	if err != nil {
		return nil, err
	}
	res.CreatedAt = r.date

	return res, nil
}

// FindByURI mock flare.ResourceRepositorier.FindByURI.
func (r *Resource) FindByURI(context.Context, string) (*flare.Resource, error) {
	return nil, nil
}

// Create mock flare.ResourceRepositorier.Create.
func (r *Resource) Create(ctx context.Context, resource *flare.Resource) error {
	return r.base.Create(ctx, resource)
}

// Delete mock flare.ResourceRepositorier.Delete.
func (r *Resource) Delete(context.Context, string) error {
	return nil
}

// NewResource return a flare.ResourceRepositorier mock.
func NewResource(options ...func(*Resource)) *Resource {
	r := &Resource{base: memory.NewResource()}

	for _, option := range options {
		option(r)
	}

	return r
}

// ResourceError set the error to be returned during calls.
func ResourceError(err error) func(*Resource) {
	return func(r *Resource) { r.err = err }
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
				Field      string `json:"field"`
				Kind       string `json:"kind"`
				DateFormat string `json:"dateFormat"`
			}
		}, 0)
		if err := json.Unmarshal(content, &resources); err != nil {
			panic(errors.Wrap(err,
				fmt.Sprintf("error during unmarshal of '%s' into '%v'", string(content), resources),
			))
		}

		for _, rawResource := range resources {
			err := r.Create(context.Background(), &flare.Resource{
				Id:        rawResource.Id,
				Addresses: rawResource.Addresses,
				Path:      rawResource.Path,
				CreatedAt: rawResource.CreatedAt,
				Change: flare.ResourceChange{
					DateFormat: rawResource.Change.DateFormat,
					Field:      rawResource.Change.Field,
					Kind:       rawResource.Change.Kind,
				},
			})
			if err != nil {
				panic(errors.Wrap(err, "error during flare.Resource persistence"))
			}
		}
	}
}
