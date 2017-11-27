// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
)

// Document implements the data layer for the document service.
type Document struct {
	client     *Client
	database   string
	collection string
}

// FindOne return the document that match the id.
func (d *Document) FindOne(ctx context.Context, id string) (*flare.Document, error) {
	return d.findOne(ctx, id, nil)
}

// FindOneWithRevision return the document that match the id and the revision.
func (d *Document) FindOneWithRevision(
	ctx context.Context, id string, revision int64,
) (*flare.Document, error) {
	return d.findOne(ctx, id, &revision)
}

// Update a given document.
func (d *Document) Update(_ context.Context, document *flare.Document) error {
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	content := d.marshal(document)
	_, err := session.DB(d.database).C(d.collection).Upsert(bson.M{
		"id":       document.ID,
		"revision": document.Revision,
	}, content)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during document '%s' update", document.ID))
	}
	return nil
}

// Delete a given document.
func (d *Document) Delete(_ context.Context, id string) error {
	return errors.New("disabled until further correct development")
}

func (d *Document) findOne(
	ctx context.Context, id string, revision *int64,
) (*flare.Document, error) {
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	query := bson.M{"id": id}
	if revision != nil {
		query["revision"] = *revision
	}

	rawResult := make(map[string]interface{})
	err := session.
		DB(d.database).
		C(d.collection).
		Find(query).
		Sort("-revision").
		One(&rawResult)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", id), notFound: true}
		}
		return nil, errors.Wrap(err, fmt.Sprintf("error during document '%s' find", id))
	}

	result, err := d.unmarshal(rawResult)
	if err != nil {
		return nil, errors.Wrap(err, "error during document unmarshal")
	}
	return result, nil
}

func (d *Document) marshal(document *flare.Document) map[string]interface{} {
	return map[string]interface{}{
		"id":         document.ID,
		"revision":   document.Revision,
		"resourceID": document.Resource.ID,
		"updatedAt":  document.UpdatedAt,
		"content":    document.Content,
	}
}

func (d *Document) unmarshal(content map[string]interface{}) (*flare.Document, error) {
	id, ok := content["id"].(string)
	if !ok {
		return nil, errors.New("missing id")
	}

	revision, ok := content["revision"].(int64)
	if !ok {
		return nil, errors.New("missing revision")
	}

	resourceID, ok := content["resourceID"].(string)
	if !ok {
		return nil, errors.New("missing resourceID")
	}

	updatedAt, ok := content["updatedAt"].(time.Time)
	if !ok {
		return nil, errors.New("missing updatedAt")
	}

	docContent, ok := content["content"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing content")
	}

	return &flare.Document{
		ID:        id,
		Revision:  revision,
		Resource:  flare.Resource{ID: resourceID},
		UpdatedAt: updatedAt,
		Content:   docContent,
	}, nil
}

func (d *Document) ensureIndex() error {
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	err := session.
		DB(d.database).
		C(d.collection).
		EnsureIndex(mgo.Index{
			Background: true,
			Unique:     true,
			Key:        []string{"id", "-revision"},
		})
	if err != nil {
		return errors.Wrap(err, "error during index creation")
	}
	return nil
}

// NewDocument returns a configured document repository.
func NewDocument(options ...func(*Document)) (*Document, error) {
	d := &Document{}
	for _, option := range options {
		option(d)
	}

	if d.client == nil {
		return nil, errors.New("invalid client")
	}
	d.collection = "documents"
	d.database = d.client.database

	if err := d.ensureIndex(); err != nil {
		return nil, err
	}
	return d, nil
}

// DocumentClient set the client to access MongoDB.
func DocumentClient(client *Client) func(*Document) {
	return func(d *Document) {
		d.client = client
	}
}
