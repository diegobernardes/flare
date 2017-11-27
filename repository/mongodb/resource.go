// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
)

const wildcard = "{*}"

type resourceEntity struct {
	Id        string               `bson:"id"`
	Addresses []string             `bson:"addresses"`
	Path      string               `bson:"path"`
	Change    resourceChangeEntity `bson:"change"`
	CreatedAt time.Time            `bson:"createdAt"`
}

type resourceChangeEntity struct {
	Field  string `bson:"field"`
	Format string `bson:"format"`
}

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
		resources []resourceEntity
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

	return r.resourceEntitySliceToFlareResourceSlice(resources), &flare.Pagination{
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

	result := &resourceEntity{}
	if err := session.DB(r.database).C(r.collection).Find(bson.M{"id": id}).One(result); err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
		}
		return nil, errors.Wrap(err, fmt.Sprintf("could not find resource '%s'", id))
	}

	return r.resourceEntityToFlareResource(result), nil
}

// FindByURI take a URI and find the resource that match.
func (r *Resource) FindByURI(_ context.Context, rawAddress string) (*flare.Resource, error) {
	address, err := url.Parse(rawAddress)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url '%s' parse", rawAddress))
	}

	query, err := r.findResourceByURI(
		[]string{fmt.Sprintf("%s://%s", address.Scheme, address.Host)},
		address.Path,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource find")
	}

	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	result := &resourceEntity{}
	err = session.DB(r.database).C(r.collection).Find(query).One(result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{
				message: fmt.Sprintf("resource not found with address '%s'", rawAddress), notFound: true,
			}
		}
		return nil, errors.Wrap(err, fmt.Sprintf(
			"error during find resource by uri '%s'", rawAddress,
		))
	}

	return r.resourceEntityToFlareResource(result), nil
}

// Create a resource.
func (r *Resource) Create(_ context.Context, res *flare.Resource) error {
	_, err := r.findResourceByURI(res.Addresses, res.Path)
	if err == nil {
		return &errMemory{message: "resource already exists", alreadyExists: true}
	}
	if err != nil {
		if nErr, ok := err.(flare.ResourceRepositoryError); ok {
			if nErr.AlreadyExists() || nErr.PathConflict() {
				return err
			}
		} else {
			return err
		}
	}

	res.CreatedAt = time.Now()
	contentChange := bson.M{"field": res.Change.Field}
	if res.Change.Format != "" {
		contentChange["format"] = res.Change.Format
	}

	content := bson.M{
		"id":           res.ID,
		"addresses":    res.Addresses,
		"path":         res.Path,
		"pathSegments": r.pathSegments(res.Path),
		"change":       contentChange,
		"createdAt":    res.CreatedAt,
	}

	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	if err := session.DB(r.database).C(r.collection).Insert(content); err != nil {
		errors.Wrap(err, "error during resource create")
	}

	return nil
}

func (r *Resource) findResourceByURI(addresses []string, path string) (bson.M, error) {
	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	segments := strings.Split(path, "/")
	segments = segments[1:]

	query := bson.M{"pathSegments": bson.M{"$size": len(segments)}}
	if len(addresses) > 1 {
		query["addresses"] = bson.M{"$in": addresses}
	} else if len(addresses) == 1 {
		query["addresses"] = addresses[0]
	}
	count := func() (int, error) { return session.DB(r.database).C(r.collection).Find(query).Count() }

	for i, segment := range segments {
		query[fmt.Sprintf("pathSegments.%d", i)] = segment

		qtd, err := count()
		if err != nil {
			return nil, errors.Wrap(err, "error during resource find")
		}

		if qtd == 0 {
			query[fmt.Sprintf("pathSegments.%d", i)] = wildcard
			qtd, err = count()
			if err != nil {
				return nil, errors.Wrap(err, "error during resource find")
			}

			if qtd == 0 {
				return nil, &errMemory{message: "resource not found", notFound: true}
			}

			if i == len(segments)-1 {
				break
			}
		} else if i == len(segments)-1 {
			break
		}
	}

	return query, nil
}

func (r *Resource) pathSegments(path string) []string {
	segments := strings.Split(path, "/")
	result := make([]string, len(segments)-1)

	for i, segment := range segments {
		if i == 0 {
			continue
		}

		if segment[0] == '{' && segment[len(segment)-1] == '}' {
			result[i-1] = wildcard
		} else {
			result[i-1] = segment
		}
	}

	return result
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

func (r *Resource) resourceEntityToFlareResource(content *resourceEntity) *flare.Resource {
	return &flare.Resource{
		ID:        content.Id,
		Addresses: content.Addresses,
		Path:      content.Path,
		CreatedAt: content.CreatedAt,
		Change: flare.ResourceChange{
			Format: content.Change.Format,
			Field:  content.Change.Field,
		},
	}
}

func (r *Resource) resourceEntitySliceToFlareResourceSlice(
	entities []resourceEntity,
) []flare.Resource {
	result := make([]flare.Resource, len(entities))
	for i, entity := range entities {
		result[i] = *r.resourceEntityToFlareResource(&entity)
	}
	return result
}

// SetSubscriptionRepository set the subscription repository.
func (r *Resource) SetSubscriptionRepository(repo flare.SubscriptionRepositorier) error {
	if repo == nil {
		return errors.New("subscriptionRepository can't be nil")
	}
	r.subscriptionRepository = repo
	return nil
}

func (r *Resource) ensureIndex() error {
	session := r.client.session()
	session.SetMode(mgo.Monotonic, true)
	defer session.Close()

	err := session.
		DB(r.database).
		C(r.collection).
		EnsureIndex(mgo.Index{
			Background: true,
			Unique:     true,
			Key:        []string{"addresses"},
		})
	if err != nil {
		return errors.Wrap(err, "error during index creation")
	}
	return nil
}

// Init configure the resource repository.
func (r *Resource) Init(options ...func(*Resource)) error {
	for _, option := range options {
		option(r)
	}

	if r.client == nil {
		return errors.New("invalid client")
	}

	if r.subscriptionRepository == nil {
		return errors.New("invalid subscription repository")
	}

	r.collection = "resources"
	r.database = r.client.database

	return r.ensureIndex()
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
