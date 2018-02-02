// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare"
)

type resourceRepositorier interface {
	flare.ResourceRepositorier
	leavePartition(ctx context.Context, id, partition string) error
	joinPartition(ctx context.Context, id string) (string, error)
}

type resource struct {
	base       flare.Resource
	partitions map[string]int
}

// Resource implements the data layer for the resource service.
type Resource struct {
	mutex          sync.RWMutex
	resources      []resource
	partitionLimit int
	repository     flare.SubscriptionRepositorier
}

func transformResources(res []resource) []flare.Resource {
	resources := make([]flare.Resource, len(res))
	for i, r := range res {
		resources[i] = r.base
	}
	return resources
}

// Find returns a list of resources.
func (r *Resource) Find(
	_ context.Context,
	pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var resp []resource
	if pagination.Offset > len(r.resources) {
		resp = r.resources
	} else if pagination.Limit+pagination.Offset > len(r.resources) {
		resp = r.resources[pagination.Offset:]
	} else {
		resp = r.resources[pagination.Offset : pagination.Offset+pagination.Limit]
	}

	return transformResources(resp), &flare.Pagination{
		Total:  len(r.resources),
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}, nil
}

// FindByID return the resource that match the id.
func (r *Resource) FindByID(ctx context.Context, id string) (*flare.Resource, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	res, err := r.findByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &res.base, nil
}

func (r *Resource) findByID(_ context.Context, id string) (*resource, error) {
	for _, resource := range r.resources {
		if resource.base.ID == id {
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
		if resource.base.ID == res.ID {
			return &errMemory{
				alreadyExists: true, message: fmt.Sprintf("already exists a resource with id '%s'", res.ID),
			}
		}

		if sliceIntersection(
			resource.base.Addresses,
			res.Addresses,
			r.normalizePath(resource.base.Path),
			r.normalizePath(res.Path),
		) {
			return &errMemory{
				message: fmt.Sprintf(
					"address+path already associated to another resource '%s'", resource.base.ID,
				),
				alreadyExists: true,
			}
		}
	}

	res.CreatedAt = time.Now()
	r.resources = append(r.resources, resource{
		base:       *res,
		partitions: make(map[string]int),
	})
	return nil
}

func (r *Resource) normalizePath(raw string) string {
	result := []string{"/"}
	for _, segment := range strings.Split(raw, "/") {
		if len(segment) >= 2 && segment[0] == '{' && segment[len(segment)-1] == '}' {
			result = append(result, "{*}")
			continue
		}
		result = append(result, segment)
	}
	return strings.Join(result, "/")
}

// Delete a given resource.
func (r *Resource) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, pagination, err := r.repository.Find(ctx, &flare.Pagination{Limit: 1}, id)
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
		if res.base.ID == id {
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

// Partitions return the partitions a given resource have.
func (r *Resource) Partitions(ctx context.Context, id string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	res, err := r.findByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var partitions []string
	for partition := range res.partitions {
		partitions = append(partitions, partition)
	}
	return partitions, nil
}

func (r *Resource) findResourcesByHost(uri *url.URL) ([]flare.Resource, error) {
	var resources []flare.Resource
	for _, resource := range r.resources {
		for _, rawAddress := range resource.base.Addresses {
			address, err := url.Parse(rawAddress)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("error during address parse '%s'", address))
			}

			if address.Host == uri.Host {
				resources = append(resources, resource.base)
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

func (r *Resource) joinPartition(ctx context.Context, id string) (string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	resource, err := r.findByID(ctx, id)
	if err != nil {
		return "", err
	}

	for key, value := range resource.partitions {
		if r.partitionLimit > value+1 {
			resource.partitions[key]++
			return key, nil
		}
	}

	partition := uuid.NewV4().String()
	resource.partitions[partition] = 1
	return partition, nil
}

func (r *Resource) leavePartition(ctx context.Context, id, partition string) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	resource, err := r.findByID(ctx, id)
	if err != nil {
		return err
	}

	for key := range resource.partitions {
		if key == partition {
			resource.partitions[key]--
			if resource.partitions[key] == 0 {
				delete(resource.partitions, key)
			}
		}
	}
	return nil
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

func (r *Resource) init(options ...func(*Resource)) {
	r.resources = make([]resource, 0)

	for _, option := range options {
		option(r)
	}
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

// ResourcePartitionLimit set the max quantity of subscriptions per partition.
func ResourcePartitionLimit(limit int) func(*Resource) {
	return func(r *Resource) {
		r.partitionLimit = limit
	}
}
