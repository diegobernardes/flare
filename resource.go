// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Resource is the base component from Flare. It is holds the data to detect the document change.
type Resource struct {
	ID        string
	Addresses []string
	Path      string
	Change    ResourceChange
	CreatedAt time.Time
}

// WildcardReplace take a string and search of wildcards to replace the value.
func (r *Resource) WildcardReplace(documentPath string) (func(string) string, error) {
	endpoint, err := url.Parse(documentPath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url parse of '%s'", documentPath))
	}
	wildcards := strings.Split(r.Path, "/")
	documentWildcards := strings.Split(endpoint.Path, "/")

	return func(value string) string {
		for i, wildcard := range wildcards {
			if wildcard == "" {
				continue
			}

			if wildcard[0] == '{' && wildcard[len(wildcard)-1] == '}' {
				value = strings.Replace(value, wildcard, documentWildcards[i], -1)
			}
		}
		return value
	}, nil
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
	FindByURI(context.Context, string) (*Resource, error)
	Create(context.Context, *Resource) error
	Delete(context.Context, string) error
}

// ResourceRepositoryError implements all the errrors the repository can return.
type ResourceRepositoryError interface {
	AlreadyExists() bool
	PathConflict() bool
	NotFound() bool
}
