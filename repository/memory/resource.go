// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memory

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Resource implements the data layer for the resource service.
type Resource struct {
	mutex                  sync.RWMutex
	resources              []flare.Resource
	subscriptionRepository flare.SubscriptionRepositorier
}

// FindAll returns a list of resources.
func (r *Resource) FindAll(
	_ context.Context,
	pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var resp []flare.Resource
	if pagination.Offset > len(r.resources) {
		resp = r.resources
	} else if pagination.Limit+pagination.Offset > len(r.resources) {
		resp = r.resources[pagination.Offset:]
	} else {
		resp = r.resources[pagination.Offset : pagination.Offset+pagination.Limit]
	}

	return resp, &flare.Pagination{
		Total:  len(r.resources),
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}, nil
}

// FindOne return the resource that match the id.
func (r *Resource) FindOne(_ context.Context, id string) (*flare.Resource, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, resource := range r.resources {
		if resource.ID == id {
			return &resource, nil
		}
	}
	return nil, &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
}

// Create a resource.
func (r *Resource) Create(_ context.Context, res *flare.Resource) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, resource := range r.resources {
		if resource.ID == res.ID {
			return &errMemory{
				alreadyExists: true, message: fmt.Sprintf("already exists a resource with id '%s'", res.ID),
			}
		}

		if sliceIntersection(resource.Addresses, res.Addresses, resource.Path, res.Path) {
			return &errMemory{
				message: fmt.Sprintf(
					"address+path already associated to another resource '%s'", resource.ID,
				),
				pathConflict: true,
			}
		}
	}

	res.CreatedAt = time.Now()
	r.resources = append(r.resources, *res)
	return nil
}

// Delete a given resource.
func (r *Resource) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, pagination, err := r.subscriptionRepository.FindAll(ctx, &flare.Pagination{Limit: 1}, id)
	if err != nil {
		return errors.Wrap(err, "error during subscription search")
	}
	if pagination.Total > 0 {
		return &errMemory{
			message:  fmt.Sprintf("there are subscriptions associated with this resource '%s'", id),
			notFound: true,
		}
	}

	for i, res := range r.resources {
		if res.ID == id {
			r.resources = append(r.resources[:i], r.resources[i+1:]...)
			return nil
		}
	}

	return &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
}

// FindByURI take a URI and find the resource that match.
func (r *Resource) FindByURI(_ context.Context, rawURI string) (*flare.Resource, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !strings.HasPrefix(rawURI, "http") {
		rawURI = "//" + rawURI
	}

	uri, err := url.Parse(rawURI)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url.Parse with '%s'", rawURI))
	}

	resources, err := r.findResourcesByHost(uri)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource search")
	}

	resource, err := r.selectResouceByHost(uri, resources)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource select")
	}
	if resource != nil {
		return resource, nil
	}
	return nil, &errMemory{
		notFound: true, message: fmt.Sprintf("could not found a resource for this uri '%s'", rawURI),
	}
}

func (r *Resource) findResourcesByHost(uri *url.URL) ([]flare.Resource, error) {
	var resources []flare.Resource
	for _, resource := range r.resources {
		for _, rawAddress := range resource.Addresses {
			address, err := url.Parse(rawAddress)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("error during address parse '%s'", address))
			}

			if address.Host == uri.Host {
				resources = append(resources, resource)
				break
			}
		}
	}
	return resources, nil
}

func (r *Resource) selectResouceByHost(
	uri *url.URL, resources []flare.Resource,
) (*flare.Resource, error) {
	segments := strings.Split(uri.Path, "/")
outer:
	for _, resourceSegment := range r.genResourceSegments(resources, len(segments)) {
		for i := 0; i < len(segments); i++ {
			segment := resourceSegment[i+1]
			if segments[i] == segment {
				continue
			} else if segment[0] == '{' && segment[len(segment)-1] == '}' {
				continue
			} else {
				continue outer
			}
		}

		for _, resource := range resources {
			if resource.ID == resourceSegment[0] {
				return &resource, nil
			}
		}
		break
	}
	return nil, nil
}

func (r *Resource) genResourceSegments(resources []flare.Resource, qtySegments int) [][]string {
	result := make([][]string, 0)

	for _, resource := range resources {
		segments := strings.Split(resource.Path, "/")
		if len(segments) != qtySegments {
			continue
		}
		result = append(result, append([]string{resource.ID}, segments...))
	}

	if len(result) > 1 {
		sort.Sort(segment(result))
	}
	return result
}

func sliceIntersection(a, b []string, a1, b1 string) bool {
	for _, aValue := range a {
		for _, bValue := range b {
			if aValue+a1 == bValue+b1 {
				return true
			}
		}
	}
	return false
}

// NewResource returns a configured resource repository.
func NewResource(options ...func(*Resource)) *Resource {
	r := &Resource{resources: make([]flare.Resource, 0)}

	for _, option := range options {
		option(r)
	}

	if r.subscriptionRepository == nil {
		r.subscriptionRepository = NewSubscription()
	}

	return r
}

// ResourceSubscriptionRepository .
func ResourceSubscriptionRepository(
	subscriptionRepository flare.SubscriptionRepositorier,
) func(*Resource) {
	return func(r *Resource) { r.subscriptionRepository = subscriptionRepository }
}

type segment [][]string

func (s segment) Len() int { return len(s) }

func (s segment) Less(i, j int) bool {
	for aux := 0; aux < len(s[i]); aux++ {
		if s[i][aux] == s[j][aux] {
			continue
		} else if s[i][aux][0] == '{' && s[i][aux][len(s[i][aux])-1] == '}' {
			return false
		} else if s[j][aux][0] == '{' && s[j][aux][len(s[j][aux])-1] == '}' {
			return true
		}
	}
	return false
}

func (s segment) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
