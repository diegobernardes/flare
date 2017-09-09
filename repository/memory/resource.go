package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/diegobernardes/flare"
)

type errResource struct {
	message       string
	alreadyExists bool
	pathConflict  bool
	notFound      bool
}

func (e *errResource) Error() string       { return e.message }
func (e *errResource) AlreadyExists() bool { return e.alreadyExists }
func (e *errResource) PathConflict() bool  { return e.pathConflict }
func (e *errResource) NotFound() bool      { return e.notFound }

// Resource implements the data layer for the Resource service.
type Resource struct {
	mutex     sync.RWMutex
	resources []flare.Resource
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
		if resource.Id == id {
			return &resource, nil
		}
	}
	return nil, &errResource{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
}

// Create a resource.
func (r *Resource) Create(_ context.Context, res *flare.Resource) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, resource := range r.resources {
		if resource.Id == res.Id {
			return &errResource{
				alreadyExists: true, message: fmt.Sprintf("already exists a resource with id '%s'", res.Id),
			}
		}

		if sliceIntersection(resource.Domains, res.Domains, resource.Path, res.Path) {
			return &errResource{
				message: fmt.Sprintf(
					"domain+path already associated to another resource '%s'", resource.Id,
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
func (r *Resource) Delete(_ context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, res := range r.resources {
		if res.Id == id {
			r.resources = append(r.resources[:i], r.resources[i+1:]...)
			return nil
		}
	}

	return &errResource{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
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
func NewResource() *Resource {
	return &Resource{resources: make([]flare.Resource, 0)}
}
