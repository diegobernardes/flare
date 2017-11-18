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
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	result := &flare.Document{}
	if err := session.DB(d.database).C(d.collection).Find(bson.M{"id": id}).One(result); err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("document '%s' not found", id), notFound: true}
		}
		return nil, errors.Wrap(err, fmt.Sprintf("error during document '%s' find", id))
	}

	return result, nil
}

// Update a given document.
func (d *Document) Update(_ context.Context, document *flare.Document) error {
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()
	document.UpdatedAt = time.Now()

	_, err := session.DB(d.database).C(d.collection).Upsert(bson.M{"id": document.Id}, document)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during document '%s' update", document.Id))
	}
	return nil
}

// Delete a given document.
func (d *Document) Delete(_ context.Context, id string) error {
	session := d.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	if err := session.DB(d.database).C(d.collection).Remove(bson.M{"id": id}); err != nil {
		if err == mgo.ErrNotFound {
			return &errMemory{message: fmt.Sprintf("document '%s' not found", id), notFound: true}
		}
		return errors.Wrap(err, fmt.Sprintf("error during document '%s' delete", id))
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
	return d, nil
}

// DocumentClient set the client to access MongoDB.
func DocumentClient(client *Client) func(*Document) {
	return func(d *Document) {
		d.client = client
	}
}
