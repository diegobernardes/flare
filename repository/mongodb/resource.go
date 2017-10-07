// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
)

// Resource implements the data layer for the resource service.
type Resource struct {
	subscriptionRepository flare.SubscriptionRepositorier
	client                 *Client
	database               string
	collection             string
}

// FindAll returns a list of resources.
func (r *Resource) FindAll(
	_ context.Context, pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	var (
		group     errgroup.Group
		resources []flare.Resource
		total     int
	)

	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	group.Go(func() error {
		totalResult, err := session.DB(r.database).C(r.collection).Find(bson.M{}).Count()
		if err != nil {
			return err
		}
		total = totalResult
		return nil
	})

	group.Go(func() error {
		q := session.
			DB(r.database).
			C(r.collection).
			Find(bson.M{}).
			Sort("createdAt").
			Limit(pagination.Limit)

		if pagination.Offset != 0 {
			q = q.Skip(pagination.Offset)
		}

		return q.All(&resources)
	})

	if err := group.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "error during MongoDB access")
	}

	return resources, &flare.Pagination{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}

// FindOne return the resource that match the id.
func (r *Resource) FindOne(_ context.Context, id string) (*flare.Resource, error) {
	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	result := &flare.Resource{}
	if err := session.DB(r.database).C(r.collection).Find(bson.M{"id": id}).One(result); err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
		}
		return result, errors.Wrap(err, fmt.Sprintf("error during resource '%s' find", id))
	}

	return result, nil
}

// FindByURI take a URI and find the resource that match.
func (r *Resource) FindByURI(_ context.Context, rawAddress string) (*flare.Resource, error) {
	address, err := url.Parse(rawAddress)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url '%s' parse", rawAddress))
	}
	parsedAddress := fmt.Sprintf("%s://%s", address.Scheme, address.Host)

	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	result := &flare.Resource{}
	err = session.
		DB(r.database).
		C(r.collection).
		Find(bson.M{"addresses": parsedAddress}).
		One(result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{
				message: fmt.Sprintf("resource not found with address '%s'", parsedAddress), notFound: true,
			}
		}
		return nil, errors.Wrap(err, fmt.Sprintf(
			"error during find resource by uri '%s'", address.String(),
		))
	}

	return result, nil
}

// Create a resource.
func (r *Resource) Create(_ context.Context, res *flare.Resource) error {
	var group errgroup.Group

	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	for _, rawAddr := range res.Addresses {
		group.Go(func(addr string) func() error {
			return func() error {
				qtd, err := session.
					DB(r.database).
					C(r.collection).
					Find(bson.M{"addresses": addr}).
					Limit(1).
					Count()
				if err != nil {
					return errors.Wrap(err, "error during resource create")
				}

				if qtd > 0 {
					return &errMemory{
						message:       fmt.Sprintf("already has a resource with the address '%s'", addr),
						alreadyExists: true,
					}
				}

				return nil
			}
		}(rawAddr))
	}

	if err := group.Wait(); err != nil {
		return err
	}

	res.CreatedAt = time.Now()
	if err := session.DB(r.database).C(r.collection).Insert(res); err != nil {
		errors.Wrap(err, "error during resource create")
	}

	return nil
}

// Delete a given resource.
func (r *Resource) Delete(_ context.Context, id string) error {
	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	if err := session.DB("flare").C("resources").Remove(bson.M{"id": id}); err != nil {
		if err == mgo.ErrNotFound {
			return &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
		}
		return errors.Wrap(err, fmt.Sprintf("error during resource '%s' delete", id))
	}

	return nil
}

// NewResource returns a configured resource repository.
func NewResource(options ...func(*Resource)) (*Resource, error) {
	r := &Resource{}
	for _, option := range options {
		option(r)
	}

	if r.client == nil {
		return nil, errors.New("invalid client")
	}

	if r.subscriptionRepository == nil {
		return nil, errors.New("invalid subscription repository")
	}

	r.collection = "resources"
	r.database = r.client.database
	return r, nil
}

// ResourceSubscriptionRepository set the repository to access the subscriptions.
func ResourceSubscriptionRepository(
	subscriptionRepository flare.SubscriptionRepositorier,
) func(*Resource) {
	return func(r *Resource) { r.subscriptionRepository = subscriptionRepository }
}

// ResourceClient set the client to access MongoDB.
func ResourceClient(client *Client) func(*Resource) {
	return func(r *Resource) {
		r.client = client
	}
}
