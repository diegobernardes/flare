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

	if doc.Resource.Change.Kind != ResourceChangeDate {
		return nil
	}

	changeFieldValue, ok := doc.ChangeFieldValue.(string)
	if !ok {
		return errors.New("invalid ChangeFieldValue, could not cast it to string")
	}

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

	return nil
}

// Newer indicates if the current document is newer then the one passed as parameter.
func (doc *Document) Newer(reference *Document) (bool, error) {
	if doc == nil {
		return true, nil
	}

	switch doc.Resource.Change.Kind {
	case ResourceChangeDate:
		return doc.newerDate(reference.ChangeFieldValue)
	case ResourceChangeInteger:
		return doc.newerInteger(doc.ChangeFieldValue, reference.ChangeFieldValue)
	case ResourceChangeString:
		return doc.newDocumentVersionString(doc.ChangeFieldValue, reference.ChangeFieldValue)
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

func (doc *Document) newerInteger(rawDocValue, rawReferenceValue interface{}) (bool, error) {
	docValue, ok := rawDocValue.(int64)
	if !ok {
		return false, fmt.Errorf("could not cast rawDocValue(%v) to integer", rawDocValue)
	}

	referenceValue, ok := rawReferenceValue.(int64)
	if !ok {
		return false, fmt.Errorf("could not cast rawReferenceValue(%v) to integer", rawReferenceValue)
	}

	return referenceValue > docValue, nil
}

func (doc *Document) newDocumentVersionString(rawDocValue, rawReferenceValue interface{},
) (bool, error) {
	docValue, ok := rawDocValue.(string)
	if !ok {
		return false, fmt.Errorf("could not cast rawDocValue(%v) to string", rawDocValue)
	}

	referenceValue, ok := rawReferenceValue.(string)
	if !ok {
		return false, fmt.Errorf("could not cast rawReferenceValue(%v) to string", rawReferenceValue)
	}

	return referenceValue > docValue, nil
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
