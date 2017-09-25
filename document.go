// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Document represents the documents from a Resource.
type Document struct {
	Id               string
	ChangeFieldValue interface{}
	Resource         Resource
	UpdatedAt        time.Time
}

// Valid indicates if the current document is valid.
func (doc *Document) Valid() error {
	if doc.Id == "" {
		return errors.New("missing Id")
	}

	if doc.ChangeFieldValue == nil {
		return errors.New("missing ChangeField")
	}

	if err := doc.Resource.Change.Valid(); err != nil {
		return errors.Wrap(err, "invalid Resource.Change")
	}

	switch doc.Resource.Change.Kind {
	case ResourceChangeDate, ResourceChangeString:
		changeFieldValue, ok := doc.ChangeFieldValue.(string)
		if !ok {
			return errors.New("invalid ChangeFieldValue, could not cast it to string")
		}

		if doc.Resource.Change.Kind == ResourceChangeDate {
			_, err := time.Parse(doc.Resource.Change.DateFormat, changeFieldValue)
			if err != nil {
				return errors.Wrap(
					err,
					fmt.Sprintf(
						"error during time.Parse with format '%s' and value '%s'",
						doc.Resource.Change.DateFormat,
						changeFieldValue,
					))
			}
		}
	}

	return nil
}

// Newer indicates if the current document is newer then the one passed as parameter.
func (doc *Document) Newer(reference *Document) (bool, error) {
	if reference == nil {
		return true, nil
	}

	switch doc.Resource.Change.Kind {
	case ResourceChangeDate:
		return doc.newerDate(reference.ChangeFieldValue)
	case ResourceChangeInteger:
		return doc.newerInteger(reference.ChangeFieldValue)
	case ResourceChangeString:
		return doc.newerString(reference.ChangeFieldValue)
	default:
		return false, errors.New("invalid change kind")
	}
}

func (doc *Document) newerDate(rawReferenceValue interface{}) (bool, error) {
	docValue, ok := doc.ChangeFieldValue.(string)
	if !ok {
		return false, fmt.Errorf(
			"could not cast the changeFieldValue '%v' to string", doc.ChangeFieldValue,
		)
	}

	referenceValue, ok := rawReferenceValue.(string)
	if !ok {
		return false, fmt.Errorf(
			"could not cast the reference changeFieldValue '%v' to string", rawReferenceValue,
		)
	}

	format := doc.Resource.Change.DateFormat
	docValueTime, err := time.Parse(format, docValue)
	if err != nil {
		return false, fmt.Errorf(
			"error during time.Parse using changeFieldValue '%s' and format '%s'",
			docValue,
			format,
		)
	}

	referenceValueTime, err := time.Parse(format, referenceValue)
	if err != nil {
		return false, fmt.Errorf(
			"error during time.Parse using reference changeFieldValue '%s' and format '%s'",
			referenceValue,
			format,
		)
	}

	return docValueTime.After(referenceValueTime), nil
}

func (doc *Document) newerInteger(rawReferenceValue interface{}) (bool, error) {
	docValue, ok := doc.ChangeFieldValue.(float64)
	if !ok {
		return false, fmt.Errorf("could not cast rawDocValue '%v' to float64", doc.ChangeFieldValue)
	}

	referenceValue, ok := rawReferenceValue.(float64)
	if !ok {
		return false, fmt.Errorf("could not cast rawReferenceValue '%v' to float64", rawReferenceValue)
	}

	return docValue > referenceValue, nil
}

func (doc *Document) newerString(rawReferenceValue interface{}) (bool, error) {
	docValue, ok := doc.ChangeFieldValue.(string)
	if !ok {
		return false, fmt.Errorf("could not cast rawDocValue(%v) to string", doc.ChangeFieldValue)
	}

	referenceValue, ok := rawReferenceValue.(string)
	if !ok {
		return false, fmt.Errorf("could not cast rawReferenceValue(%v) to string", rawReferenceValue)
	}

	return docValue > referenceValue, nil
}

// DocumentRepositorier used to interact with Document data storage.
type DocumentRepositorier interface {
	FindOne(ctx context.Context, id string) (*Document, error)
	Update(context.Context, *Document) error
	Delete(ctx context.Context, id string) error
}

// DocumentRepositoryError implements all the errrors the repository can return.
type DocumentRepositoryError interface {
	NotFound() bool
}
