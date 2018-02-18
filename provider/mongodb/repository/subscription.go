// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

type subscriptionEntity struct {
	ID         string                     `bson:"id"`
	ResourceID string                     `bson:"resourceID"`
	Endpoint   subscriptionEndpointEntity `bson:"endpoint"`
	Delivery   subscriptionDeliveryEntity `bson:"delivery"`
	Partition  string                     `bson:"partition"`
	Data       map[string]interface{}     `bson:"data"`
	Content    subscriptionContentEntity  `bson:"content"`
	CreatedAt  time.Time                  `bson:"createdAt"`
}

type subscriptionEndpointEntity struct {
	URLS    []subscriptionURLEntity               `bson:"urls,omitempty"`
	Method  string                                `bson:"method,omitempty"`
	Headers http.Header                           `bson:"headers,omitempty"`
	Action  map[string]subscriptionEndpointEntity `bson:"action,omitempty"`
}

type subscriptionDeliveryEntity struct {
	Success []int `bson:"success"`
	Discard []int `bson:"discard"`
}

type subscriptionContentEntity struct {
	Document bool `bson:"document"`
	Envelope bool `bson:"envelope"`
}

type subscriptionURLEntity struct {
	Scheme string `bson:"scheme"`
	Host   string `bson:"host"`
	Path   string `bson:"path"`
	Action string `bson:"action,omitempty"`
}

// Subscription implements the data layer for the subscription service.
type Subscription struct {
	resourceRepository resourceRepositorier
	documentRepository flare.DocumentRepositorier
	client             *mongodb.Client
	database           string
	collection         string
	collectionTrigger  string
}

// Find returns a list of subscriptions.
func (s *Subscription) Find(
	_ context.Context, pagination *flare.Pagination, id string,
) ([]flare.Subscription, *flare.Pagination, error) {
	var (
		group         errgroup.Group
		subscriptions []subscriptionEntity
		total         int
	)

	group.Go(func() error {
		session := s.client.Session()
		defer session.Close()

		totalResult, err := session.DB(s.database).C(s.collection).Find(bson.M{"resourceID": id}).Count()
		if err != nil {
			return err
		}
		total = totalResult
		return nil
	})

	group.Go(func() error {
		session := s.client.Session()
		defer session.Close()

		q := session.
			DB(s.database).
			C(s.collection).
			Find(bson.M{"resourceID": id}).
			Sort("createdAt").
			Limit(pagination.Limit)
		if pagination.Offset != 0 {
			q = q.Skip(pagination.Offset)
		}

		return q.All(&subscriptions)
	})

	if err := group.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "error during MongoDB access")
	}

	return s.marshalSlice(subscriptions), &flare.Pagination{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
		Total:  total,
	}, nil
}

// FindByID return the Subscription that match the id.
func (s *Subscription) FindByID(
	_ context.Context, resourceId, id string,
) (*flare.Subscription, error) {
	session := s.client.Session()
	defer session.Close()

	result := &subscriptionEntity{}
	err := session.DB(s.database).C(s.collection).Find(bson.M{"id": id}).One(result)
	if err == mgo.ErrNotFound {
		return nil, &errMemory{message: fmt.Sprintf(
			"subscription '%s' at resource '%s' not found", id, resourceId,
		), notFound: true}
	}
	return s.marshal(result),
		errors.Wrap(err, "error during subscription search")
}

// Create a subscription.
func (s *Subscription) Create(ctx context.Context, subscription *flare.Subscription) error {
	session := s.client.Session()
	defer session.Close()

	var queryEndpoint []bson.M
	if subscription.Endpoint.URL != nil {
		queryEndpoint = append(
			queryEndpoint,
			bson.M{
				"endpoint.urls.scheme": subscription.Endpoint.URL.Scheme,
				"endpoint.urls.host":   subscription.Endpoint.URL.Host,
				"endpoint.urls.path":   subscription.Endpoint.URL.Path,
			},
		)
	}

	for action, endpoint := range subscription.Endpoint.Action {
		if endpoint.URL == nil {
			continue
		}

		queryEndpoint = append(
			queryEndpoint,
			bson.M{
				"endpoint.urls.scheme": endpoint.URL.Scheme,
				"endpoint.urls.host":   endpoint.URL.Host,
				"endpoint.urls.path":   endpoint.URL.Path,
				"endpoint.urls.action": action,
			},
		)
	}

	sub := &subscriptionEntity{}
	err := session.DB(s.database).C(s.collection).Find(bson.M{
		"resourceID": subscription.Resource.ID,
		"$or":        queryEndpoint,
	}).Select(bson.M{"id": 1}).One(sub)
	if err == nil {
		return &errMemory{
			message:       fmt.Sprintf("already has a subscription '%s' with this endpoint", sub.ID),
			alreadyExists: true,
		}
	}
	if err != nil && err != mgo.ErrNotFound {
		return errors.Wrap(err, "error during subscription search")
	}

	partition, err := s.resourceRepository.joinPartition(ctx, subscription.Resource.ID)
	if err != nil {
		return errors.Wrap(err, "error during subscription join resource partition")
	}
	subscription.Partition = partition

	subscription.CreatedAt = time.Now()
	err = session.DB(s.database).C(s.collection).Insert(s.unmarshal(subscription))
	return errors.Wrap(err, "error during subscription create")
}

// FindByPartition find all subscriptions that belongs to a given partition.
func (s *Subscription) FindByPartition(
	_ context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	chanSubscriptions := make(chan flare.Subscription)
	chanError := make(chan error)

	go func() {
		session := s.client.Session()
		defer session.Close()

		iter := session.
			DB(s.database).
			C(s.collection).
			Find(bson.M{"partition": partition, "resourceID": resourceID}).
			Iter()
		defer func() { _ = iter.Close() }()

		for {
			rawResult := &subscriptionEntity{}
			next := iter.Next(rawResult)
			if !next {
				if iter.Timeout() {
					continue
				}

				if err := iter.Err(); err != nil {
					chanError <- errors.Wrap(err, "error during subscription fetch")
				}
				break
			}

			result := s.marshal(rawResult)
			chanSubscriptions <- *result
		}

		close(chanSubscriptions)
	}()

	return chanSubscriptions, chanError, nil
}

// Delete a given subscription.
func (s *Subscription) Delete(ctx context.Context, resourceId, id string) error {
	session := s.client.Session()
	defer session.Close()

	subscription, err := s.FindByID(ctx, resourceId, id)
	if err != nil {
		return err
	}

	c := session.DB(s.database).C(s.collection)
	if err = c.Remove(bson.M{"id": id, "resourceID": resourceId}); err != nil {
		if err == mgo.ErrNotFound {
			return &errMemory{message: fmt.Sprintf(
				"subscription '%s' at resource '%s' not found", id, resourceId,
			), notFound: true}
		}
	}

	if err = s.resourceRepository.leavePartition(ctx, resourceId, subscription.Partition); err != nil {
		return err
	}
	return nil
}

// Trigger process the update on a document.
func (s *Subscription) Trigger(
	ctx context.Context,
	kind string,
	doc *flare.Document,
	sub *flare.Subscription,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	session := s.client.Session()
	defer session.Close()

	subscription := &subscriptionEntity{}
	err := session.
		DB(s.database).
		C(s.collection).
		Find(bson.M{"resourceID": doc.Resource.ID, "id": sub.ID}).
		One(subscription)
	if err != nil {
		return errors.Wrap(err, "error while subscription search")
	}

	resource, err := s.resourceRepository.FindByID(ctx, doc.Resource.ID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during resource '%s' find", doc.Resource.ID))
	}

	doc, err = s.documentRepository.FindByID(ctx, doc.ID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during document '%s' find", doc.ID))
	}
	doc.Resource = *resource

	return s.triggerProcess(ctx, s.marshal(subscription), doc, kind, fn)
}

func (s *Subscription) loadReferenceDocument(
	session *mgo.Session,
	subs *flare.Subscription,
	doc *flare.Document,
) (*flare.Document, error) {
	content := make(map[string]interface{})
	err := session.
		DB(s.database).
		C(s.collectionTrigger).
		Find(bson.M{"subscriptionID": subs.ID, "document.id": doc.ID}).
		One(&content)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "error during search")
	}

	return &flare.Document{
		ID:       doc.ID,
		Revision: content["document"].(map[string]interface{})["revision"].(int64),
	}, nil
}

func (s *Subscription) newEntry(
	groupCtx context.Context,
	kind string,
	session *mgo.Session,
	subs *flare.Subscription,
	doc *flare.Document,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	if kind == flare.SubscriptionTriggerDelete {
		return nil
	}

	err := s.upsertSubscriptionTrigger(session, subs, doc)
	if err != nil {
		return errors.Wrap(err, "error during document upsert")
	}

	if err = fn(groupCtx, doc, subs, flare.SubscriptionTriggerCreate); err != nil {
		return errors.Wrap(err, "error during document subscription processing")
	}
	return nil
}

func (s *Subscription) triggerProcessDelete(
	groupCtx context.Context,
	kind string,
	session *mgo.Session,
	subs *flare.Subscription,
	doc *flare.Document,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	if err := fn(groupCtx, doc, subs, flare.SubscriptionTriggerDelete); err != nil {
		return errors.Wrap(err, "error during document subscription processing")
	}

	err := session.
		DB(s.database).
		C(s.collectionTrigger).
		Remove(bson.M{"subscriptionID": subs.ID, "document.id": doc.ID})
	if err != nil {
		return errors.Wrap(err, "error during subscriptionTriggers delete")
	}

	return nil
}

func (s *Subscription) upsertSubscriptionTrigger(
	session *mgo.Session,
	subs *flare.Subscription,
	doc *flare.Document,
) error {
	_, err := session.
		DB(s.database).
		C(s.collectionTrigger).
		Upsert(
			bson.M{"subscriptionID": subs.ID, "document.id": doc.ID},
			bson.M{"subscriptionID": subs.ID, "document": bson.M{
				"id":        doc.ID,
				"revision":  doc.Revision,
				"updatedAt": time.Now(),
			}},
		)
	if err != nil {
		return errors.Wrap(err, "error during update subscriptionTriggers")
	}
	return nil
}

func (s *Subscription) triggerProcess(
	groupCtx context.Context,
	subs *flare.Subscription,
	doc *flare.Document,
	kind string,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	session := s.client.Session()
	defer session.Close()

	referenceDocument, err := s.loadReferenceDocument(session, subs, doc)
	if err != nil {
		return errors.Wrap(err, "error during reference document search")
	}

	if referenceDocument == nil {
		return s.newEntry(groupCtx, kind, session, subs, doc, fn)
	}
	referenceDocument.Resource = subs.Resource

	if kind == flare.SubscriptionTriggerDelete {
		return s.triggerProcessDelete(groupCtx, kind, session, subs, doc, fn)
	}

	if newer := doc.Newer(referenceDocument); !newer {
		return nil
	}

	if err = fn(groupCtx, doc, subs, flare.SubscriptionTriggerUpdate); err != nil {
		return errors.Wrap(err, "error during document subscription processing")
	}

	if err = s.upsertSubscriptionTrigger(session, subs, doc); err != nil {
		return errors.Wrap(err, "error during update subscriptionTriggers")
	}

	return nil
}

func (s *Subscription) unmarshal(entity *flare.Subscription) subscriptionEntity {
	return subscriptionEntity{
		ID:         entity.ID,
		ResourceID: entity.Resource.ID,
		Data:       entity.Data,
		CreatedAt:  entity.CreatedAt,
		Partition:  entity.Partition,
		Content: subscriptionContentEntity{
			Document: entity.Content.Document,
			Envelope: entity.Content.Envelope,
		},
		Delivery: subscriptionDeliveryEntity{
			Success: entity.Delivery.Success,
			Discard: entity.Delivery.Discard,
		},
		Endpoint: s.unmarshalEndpoint(entity.Endpoint),
	}
}

func (s *Subscription) unmarshalEndpoint(
	rawEntity flare.SubscriptionEndpoint,
) subscriptionEndpointEntity {
	entity := subscriptionEndpointEntity{
		Method:  rawEntity.Method,
		Headers: rawEntity.Headers,
	}

	if len(rawEntity.Action) > 0 {
		entity.Action = make(map[string]subscriptionEndpointEntity)
	}

	var urls []subscriptionURLEntity
	if rawEntity.URL != nil {
		urls = append(urls, subscriptionURLEntity{
			Scheme: rawEntity.URL.Scheme,
			Host:   rawEntity.URL.Host,
			Path:   rawEntity.URL.Path,
		})
	}

	for action, endpoint := range rawEntity.Action {
		entity.Action[action] = subscriptionEndpointEntity{
			Method:  endpoint.Method,
			Headers: endpoint.Headers,
		}

		if endpoint.URL == nil {
			continue
		}

		urls = append(urls, subscriptionURLEntity{
			Scheme: endpoint.URL.Scheme,
			Host:   endpoint.URL.Host,
			Path:   endpoint.URL.Path,
			Action: action,
		})
	}

	entity.URLS = urls
	return entity
}

func (s *Subscription) marshal(entity *subscriptionEntity) *flare.Subscription {
	return &flare.Subscription{
		ID:        entity.ID,
		Endpoint:  s.marshalEndpoint(entity.Endpoint),
		Resource:  flare.Resource{ID: entity.ResourceID},
		Partition: entity.Partition,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		Delivery: flare.SubscriptionDelivery{
			Success: entity.Delivery.Success,
			Discard: entity.Delivery.Discard,
		},
		Content: flare.SubscriptionContent{
			Document: entity.Content.Document,
			Envelope: entity.Content.Envelope,
		},
	}
}

func (s *Subscription) marshalEndpoint(
	rawEntity subscriptionEndpointEntity,
) flare.SubscriptionEndpoint {
	fetch := func(action string) *url.URL {
		for _, u := range rawEntity.URLS {
			if u.Action == action {
				return &url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path}
			}
		}
		return nil
	}

	entity := flare.SubscriptionEndpoint{
		URL:     fetch(""),
		Method:  rawEntity.Method,
		Headers: rawEntity.Headers,
	}

	if len(rawEntity.Action) > 0 {
		entity.Action = make(map[string]flare.SubscriptionEndpoint)
	}

	for action, value := range rawEntity.Action {
		entity.Action[action] = flare.SubscriptionEndpoint{
			URL:     fetch(action),
			Method:  value.Method,
			Headers: value.Headers,
		}
	}

	return entity
}

func (s *Subscription) marshalSlice(entities []subscriptionEntity) []flare.Subscription {
	result := make([]flare.Subscription, len(entities))
	for i, entity := range entities {
		result[i] = *s.marshal(&entity)
	}
	return result
}

func (s *Subscription) ensureIndex() error {
	session := s.client.Session()
	defer session.Close()

	indexes := []struct {
		index      mgo.Index
		collection string
	}{
		{
			mgo.Index{
				Background: true,
				Unique:     true,
				Key:        []string{"id", "resourceID"},
			},
			s.collection,
		},
		{
			mgo.Index{
				Background: true,
				Unique:     true,
				Key:        []string{"subscriptionID", "document.id"},
			},
			s.collectionTrigger,
		},
	}

	for _, index := range indexes {
		err := session.
			DB(s.database).
			C(index.collection).
			EnsureIndex(index.index)
		if err != nil {
			return errors.Wrap(err, "error during index creation")
		}
	}

	return nil
}

func (s *Subscription) init() error {
	if s.client == nil {
		return errors.New("invalid client")
	}

	if s.resourceRepository == nil {
		return errors.New("invalid resource repository")
	}

	if s.documentRepository == nil {
		return errors.New("invalid document repository")
	}

	s.collection = "subscriptions"
	s.collectionTrigger = "subscriptionTriggers"
	s.database = s.client.Database

	return nil
}
