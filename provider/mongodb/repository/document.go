// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	infraURL "github.com/diegobernardes/flare/infra/url"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

type documentEntity struct {
	ID         documentIDEntity       `bson:"id"`
	Revision   int64                  `bson:"revision"`
	ResourceID string                 `bson:"resourceID"`
	Content    map[string]interface{} `bson:"content"`
	UpdatedAt  time.Time              `bson:"updatedAt"`
}

type documentIDEntity struct {
	Scheme string `bson:"scheme"`
	Host   string `bson:"host"`
	Path   string `bson:"path"`
}

// Document implements the data layer for the document service.
type Document struct {
	client     *mongodb.Client
	database   string
	collection string
}

// FindByID return the document that match the id.
func (d *Document) FindByID(ctx context.Context, id url.URL) (*flare.Document, error) {
	session := d.client.Session()
	defer session.Close()

	var rawResult documentEntity
	err := session.
		DB(d.database).
		C(d.collection).
		Find(bson.M{
			"id.scheme": id.Scheme,
			"id.host":   id.Host,
			"id.path":   id.Path,
		}).
		Sort("-revision").
		One(&rawResult)
	if err != nil {
		ids, err := infraURL.String(id)
		if err != nil {
			return nil, errors.Wrap(err, "error during id transform to string")
		}

		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", ids), notFound: true}
		}
		return nil, errors.Wrap(err, fmt.Sprintf("error during document '%s' find", ids))
	}
	return d.unmarshal(rawResult), nil
}

// Update a given document.
func (d *Document) Update(_ context.Context, document *flare.Document) error {
	session := d.client.Session()
	defer session.Close()

	content := d.marshal(document)
	_, err := session.DB(d.database).C(d.collection).Upsert(bson.M{
		"id.scheme": document.ID.Scheme,
		"id.host":   document.ID.Host,
		"id.path":   document.ID.Path,
		"revision":  document.Revision,
	}, content)
	if err != nil {
		id, err := infraURL.String(document.ID)
		if err != nil {
			return errors.Wrap(err, "error during id transform to string")
		}

		return errors.Wrap(err, fmt.Sprintf("error during document '%s' update", id))
	}
	return nil
}

// Delete a given document.
func (d *Document) Delete(_ context.Context, id url.URL) error {
	session := d.client.Session()
	defer session.Close()

	return session.DB(d.database).C(d.collection).Update(bson.M{
		"scheme": id.Scheme,
		"host":   id.Host,
		"path":   id.Path,
	}, bson.M{"deleted": true})
}

// DeleteByResourceID delete all the documents from a given resource.
func (d *Document) DeleteByResourceID(ctx context.Context, id string) error {
	session := d.client.Session()
	defer session.Close()

	status, err := session.DB(d.database).C(d.collection).RemoveAll(bson.M{"resourceID": id})
	if err != nil {
		return err
	}

	if status.Matched != status.Removed {
		return fmt.Errorf(
			"could not delete all the documents, matched: %d, deleted: %d", status.Matched, status.Removed,
		)
	}
	return nil
}

func (d *Document) marshal(document *flare.Document) documentEntity {
	return documentEntity{
		ID: documentIDEntity{
			Scheme: document.ID.Scheme,
			Host:   document.ID.Host,
			Path:   document.ID.Path,
		},
		Revision:   document.Revision,
		ResourceID: document.Resource.ID,
		Content:    document.Content,
		UpdatedAt:  document.UpdatedAt,
	}
}

func (d *Document) unmarshal(rawResult documentEntity) *flare.Document {
	return &flare.Document{
		ID:        url.URL{Scheme: rawResult.ID.Scheme, Host: rawResult.ID.Host, Path: rawResult.ID.Path},
		Revision:  rawResult.Revision,
		Resource:  flare.Resource{ID: rawResult.ResourceID},
		Content:   rawResult.Content,
		UpdatedAt: rawResult.UpdatedAt,
	}
}

func (d *Document) deleteOlder(ctx context.Context, doc *flare.Document) error {
	session := d.client.Session()
	defer session.Close()

	info, err := session.DB(d.database).C(d.collection).RemoveAll(bson.M{
		"id.scheme": doc.ID.Scheme,
		"id.host":   doc.ID.Host,
		"id.path":   doc.ID.Path,
		"revision": bson.M{
			"$lt": doc.Revision,
		},
	})
	if err != nil {
		panic(err)
	}

	if info.Matched != info.Removed {
		panic("deu merda")
	}

	return nil
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
			Key:        []string{"id.scheme", "id.host", "id.path", "-revision"},
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
