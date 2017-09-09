package flare

import (
	"context"
	"errors"
	"time"
)

// Resource is the base component from Flare. It is holds the data to detect the document change.
type Resource struct {
	Id        string
	Domains   []string
	Path      string
	Change    ResourceChange
	CreatedAt time.Time
}

// The types of value Flare support to detect document change.
const (
	ResourceChangeInteger = "integer"
	ResourceChangeString  = "string"
	ResourceChangeDate    = "date"
)

// ResourceChange holds the information to detect document change.
type ResourceChange struct {
	Field      string
	Kind       string
	DateFormat string
}

// Valid indicates if the current resourceChange is valid.
func (rc *ResourceChange) Valid() error {
	if rc.Field == "" {
		return errors.New("blank field")
	}

	if rc.Kind == "" {
		return errors.New("blank kind")
	}

	if rc.Kind == ResourceChangeDate && rc.DateFormat == "" {
		return errors.New("blank dateFormat")
	}
	return nil
}

// ResourceRepositorier is used to interact with Resource repository.
type ResourceRepositorier interface {
	FindAll(context.Context, *Pagination) ([]Resource, *Pagination, error)
	FindOne(context.Context, string) (*Resource, error)
	Create(context.Context, *Resource) error
	Delete(context.Context, string) error
}

// ResourceRepositoryError implements all the errrors the repository can return.
type ResourceRepositoryError interface {
	AlreadyExists() bool
	PathConflict() bool
	NotFound() bool
}
