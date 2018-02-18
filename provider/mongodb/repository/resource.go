// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"golang.org/x/sync/errgroup"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

const wildcard = "{*}"

type resourceRepositorier interface {
	flare.ResourceRepositorier
	leavePartition(ctx context.Context, id, partition string) error
	joinPartition(ctx context.Context, id string) (string, error)
}

type resourceEntity struct {
	ID        string                 `bson:"_id"`
	Endpoint  resourceEndpointEntity `bson:"endpoint"`
	Change    resourceChangeEntity   `bson:"change"`
	Partition map[string]int         `bson:"partitions,omitempty"`
	CreatedAt time.Time              `bson:"createdAt"`
}

type resourceEndpointEntity struct {
	Scheme       string   `bson:"scheme"`
	Host         string   `bson:"host"`
	Path         string   `bson:"path"`
	PathSegments []string `bson:"pathSegments,omitempty"`
}

type resourceChangeEntity struct {
	Field  string `bson:"field"`
	Format string `bson:"format,omitempty"`
}

// Resource implements the data layer for the resource service.
type Resource struct {
	subscriptionRepository flare.SubscriptionRepositorier
	client                 *mongodb.Client
	database               string
	collection             string
	partitionLimit         int
}

// Partitions return the list of partitions of a resource.
func (r *Resource) Partitions(ctx context.Context, id string) ([]string, error) {
	session := r.client.Session()
	defer session.Close()

	var result struct {
		Partition map[string]int `bson:"partitions"`
	}

	err := session.
		DB(r.database).
		C(r.collection).
		Find(bson.M{"_id": id}).
		Select(bson.M{"partitions": 1}).
		One(&result)
	if err != nil {
		return nil, errors.Wrap(err, "error during fetch partitions")
	}

	var content []string
	for key := range result.Partition {
		content = append(content, key)
	}

	return content, nil
}

func (r *Resource) leavePartition(ctx context.Context, id, partition string) error {
	session := r.client.Session()
	defer session.Close()

	err := session.
		DB(r.database).
		C(r.collection).
		Update(bson.M{"_id": id}, bson.M{"$inc": bson.M{fmt.Sprintf("partitions.%s", partition): -1}})
	if err != nil {
		return errors.Wrap(err, "error during leave partition")
	}
	return nil
}

func (r *Resource) joinPartition(ctx context.Context, id string) (string, error) {
	resource, err := r.findByID(ctx, id)
	if err != nil {
		return "", err
	}

	var partition string
	for key, value := range resource.Partition {
		if r.partitionLimit > value {
			partition = key
			break
		}
	}
	if partition == "" {
		partition = uuid.NewV4().String()
	}

	session := r.client.Session()
	defer session.Close()

	err = session.
		DB(r.database).
		C(r.collection).
		Update(bson.M{"_id": id}, bson.M{"$inc": bson.M{fmt.Sprintf("partitions.%s", partition): 1}})
	if err != nil {
		return "", errors.Wrap(err, "error during join partition")
	}
	return partition, nil
}

// Find returns a list of resources.
func (r *Resource) Find(
	_ context.Context, pagination *flare.Pagination,
) ([]flare.Resource, *flare.Pagination, error) {
	var (
		group     errgroup.Group
		resources []resourceEntity
		total     int
	)

	session := r.client.Session()
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

// FindByID return the resource that match the id.
func (r *Resource) FindByID(ctx context.Context, id string) (*flare.Resource, error) {
	resource, err := r.findByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.resourceEntityToFlareResource(resource), nil
}

// FindByURI take a URI and find the resource that match.
func (r *Resource) FindByURI(_ context.Context, rawAddress string) (*flare.Resource, error) {
	address, err := url.Parse(rawAddress)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error during url '%s' parse", rawAddress))
	}

	query, err := r.findResourceByURI(*address)
	if err != nil {
		return nil, errors.Wrap(err, "error during resource find")
	}

	session := r.client.Session()
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
	exists, err := r.existsResourceByID(res.ID)
	if err != nil {
		return err
	}
	if exists {
		return &errMemory{message: "resource already exists", alreadyExists: true}
	}

	_, err = r.findResourceByURI(res.Endpoint)
	if err == nil {
		return &errMemory{message: "resource already exists", alreadyExists: true}
	}
	if err != nil {
		if nErr, ok := err.(flare.ResourceRepositoryError); ok {
			if nErr.AlreadyExists() {
				return err
			}
		} else {
			return err
		}
	}

	res.CreatedAt = time.Now()
	content := resourceEntity{
		ID: res.ID,
		Endpoint: resourceEndpointEntity{
			Scheme:       res.Endpoint.Scheme,
			Host:         res.Endpoint.Host,
			Path:         res.Endpoint.Path,
			PathSegments: r.pathSegments(res.Endpoint.Path),
		},
		Change: resourceChangeEntity{
			Field:  res.Change.Field,
			Format: res.Change.Format,
		},
		CreatedAt: res.CreatedAt,
	}

	session := r.client.Session()
	defer session.Close()

	if err := session.DB(r.database).C(r.collection).Insert(content); err != nil {
		return errors.Wrap(err, "error during resource create")
	}

	return nil
}

func (r *Resource) findByID(_ context.Context, id string) (*resourceEntity, error) {
	session := r.client.Session()
	defer session.Close()

	result := &resourceEntity{}
	if err := session.DB(r.database).C(r.collection).Find(bson.M{"_id": id}).One(result); err != nil {
		if err == mgo.ErrNotFound {
			return nil, &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
		}
		return nil, errors.Wrap(err, fmt.Sprintf("could not find resource '%s'", id))
	}

	return result, nil
}

func (r *Resource) findResourceByURI(endpoint url.URL) (bson.M, error) {
	session := r.client.Session()
	defer session.Close()

	segments := strings.Split(endpoint.Path, "/")
	segments = segments[1:]

	query := bson.M{
		"endpoint.pathSegments": bson.M{"$size": len(segments)},
		"endpoint.host":         endpoint.Host,
		"endpoint.scheme":       endpoint.Scheme,
	}
	count := func() (int, error) { return session.DB(r.database).C(r.collection).Find(query).Count() }

	for i, segment := range segments {
		query[fmt.Sprintf("endpoint.pathSegments.%d", i)] = segment

		qtd, err := count()
		if err != nil {
			return nil, errors.Wrap(err, "error during resource find")
		}

		if qtd == 0 {
			query[fmt.Sprintf("endpoint.pathSegments.%d", i)] = wildcard
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

func (r *Resource) existsResourceByID(id string) (bool, error) {
	session := r.client.Session()
	defer session.Close()

	count, err := session.DB(r.database).C(r.collection).Find(bson.M{"_id": id}).Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Delete a given resource.
func (r *Resource) Delete(_ context.Context, id string) error {
	session := r.client.Session()
	defer session.Close()

	if err := session.DB(r.database).C(r.collection).Remove(bson.M{"_id": id}); err != nil {
		if err == mgo.ErrNotFound {
			return &errMemory{message: fmt.Sprintf("resource '%s' not found", id), notFound: true}
		}
		return errors.Wrap(err, fmt.Sprintf("error during resource '%s' delete", id))
	}

	return nil
}

func (r *Resource) resourceEntityToFlareResource(content *resourceEntity) *flare.Resource {
	return &flare.Resource{
		ID: content.ID,
		Endpoint: url.URL{
			Scheme: content.Endpoint.Scheme,
			Host:   content.Endpoint.Host,
			Path:   content.Endpoint.Path,
		},
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
	session := r.client.Session()
	defer session.Close()

	err := session.
		DB(r.database).
		C(r.collection).
		EnsureIndex(mgo.Index{
			Background: true,
			Unique:     true,
			Key: []string{
				"endpoint.scheme",
				"endpoint.host",
				"endpoint.path",
				"endpoint.pathSegments",
			},
		})
	if err != nil {
		return errors.Wrap(err, "error during index creation")
	}
	return nil
}

func (r *Resource) init(options ...func(*Resource)) error {
	for _, option := range options {
		option(r)
	}

	if r.client == nil {
		return errors.New("invalid client")
	}

	if r.subscriptionRepository == nil {
		return errors.New("invalid subscription repository")
	}

	if r.partitionLimit == 0 {
		return errors.New("invalid partition limit")
	}

	r.collection = "resources"
	r.database = r.client.Database

	return nil
}

// ResourcePartitionLimit set the max quantity of entries per partition.
func ResourcePartitionLimit(partition int) func(*Resource) {
	return func(r *Resource) { r.partitionLimit = partition }
}
