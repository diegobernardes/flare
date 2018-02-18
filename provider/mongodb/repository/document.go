// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

type documentEntity struct {
	ID         string                 `bson:"id"`
	Revision   int64                  `bson:"revision"`
	ResourceID string                 `bson:"resourceID"`
	Content    map[string]interface{} `bson:"content"`
	UpdatedAt  time.Time              `bson:"updatedAt"`
}

// Document implements the data layer for the document service.
type Document struct {
	client     *mongodb.Client
	database   string
	collection string
}

// FindByID return the document that match the id.
func (d *Document) FindByID(ctx context.Context, id string) (*flare.Document, error) {
	return d.findByIDAndRevision(ctx, id, nil)
}

// FindByIDAndRevision return the document that match the id and the revision.
func (d *Document) FindByIDAndRevision(
	ctx context.Context, id string, revision int64,
) (*flare.Document, error) {
	return d.findByIDAndRevision(ctx, id, &revision)
}

// Update a given document.
func (d *Document) Update(_ context.Context, document *flare.Document) error {
	session := d.client.Session()
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
	session := d.client.Session()
	defer session.Close()

	return session.DB(d.database).C(d.collection).Update(bson.M{"id": id}, bson.M{"deleted": true})
}

func (d *Document) findByIDAndRevision(
	ctx context.Context, id string, revision *int64,
) (*flare.Document, error) {
	session := d.client.Session()
	defer session.Close()

	query := bson.M{"id": id}
	if revision != nil {
		query["revision"] = *revision
	}

	var rawResult documentEntity
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
	return d.unmarshal(rawResult), nil
}

func (d *Document) marshal(document *flare.Document) documentEntity {
	return documentEntity{
		ID:         document.ID,
		Revision:   document.Revision,
		ResourceID: document.Resource.ID,
		Content:    document.Content,
		UpdatedAt:  document.UpdatedAt,
	}
}

func (d *Document) unmarshal(rawResult documentEntity) *flare.Document {
	return &flare.Document{
		ID:        rawResult.ID,
		Revision:  rawResult.Revision,
		Resource:  flare.Resource{ID: rawResult.ResourceID},
		Content:   rawResult.Content,
		UpdatedAt: rawResult.UpdatedAt,
	}
}

func (d *Document) ensureIndex() error {
	session := d.client.Session()
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

func (d *Document) init() error {
	if d.client == nil {
		return errors.New("invalid client")
	}
	d.collection = "documents"
	d.database = d.client.Database
	return nil
}
