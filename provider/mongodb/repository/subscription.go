// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

// Subscription implements the data layer for the subscription service.
type Subscription struct {
	resourceRepository resourceRepositorier
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
		subscriptions []flare.Subscription
		total         int
	)

	group.Go(func() error {
		session := s.client.Session()
		defer session.Close()

		totalResult, err := session.DB(s.database).C(s.collection).Find(bson.M{"resource.id": id}).Count()
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
			Find(bson.M{"resource.id": id}).
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

	return subscriptions, &flare.Pagination{
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

	result := &flare.Subscription{}
	err := session.DB(s.database).C(s.collection).Find(bson.M{"id": id}).One(result)
	if err == mgo.ErrNotFound {
		return nil, &errMemory{message: fmt.Sprintf(
			"subscription '%s' at resource '%s' not found", id, resourceId,
		), notFound: true}
	}
	return result, errors.Wrap(err, "error during subscription search")
}

// Create a subscription.
func (s *Subscription) Create(ctx context.Context, subscription *flare.Subscription) error {
	session := s.client.Session()
	defer session.Close()

	resourceEntity := &resourceEntity{}
	err := session.DB(s.database).C(s.collection).Find(bson.M{
		"resource.id":  subscription.Resource.ID,
		"endpoint.url": subscription.Endpoint.URL.String(),
	}).One(resourceEntity)
	if err == nil {
		return &errMemory{
			message: fmt.Sprintf(
				"already has a subscription '%s' with this endpoint", resourceEntity.Id,
			),
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
	return errors.Wrap(
		session.DB(s.database).C(s.collection).Insert(subscription),
		"error during subscription create",
	)
}

// FindByPartition find all subscriptions that belongs to a given partition.
func (s *Subscription) FindByPartition(
	_ context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	session := s.client.Session()

	chanSubscriptions := make(chan flare.Subscription)
	chanError := make(chan error)

	go func() {
		defer session.Close()
		iter := session.
			DB(s.database).
			C(s.collection).
			Find(bson.M{"partition": partition, "resource.id": resourceID}).
			Iter()
		defer func() { _ = iter.Close() }()

		for {
			var result flare.Subscription
			next := iter.Next(&result)
			if !next {
				if iter.Timeout() {
					continue
				}

				if err := iter.Err(); err != nil {
					chanError <- errors.Wrap(err, "error during subscription fetch")
				}
				break
			}

			chanSubscriptions <- result
		}
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
	if err = c.Remove(bson.M{"id": id, "resource.id": resourceId}); err != nil {
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

	subscription := &flare.Subscription{}
	err := session.
		DB(s.database).
		C(s.collection).
		Find(bson.M{"resource.id": doc.Resource.ID}).
		One(subscription)
	if err != nil {
		return errors.Wrap(err, "error while subscription search")
	}

	resource, err := s.resourceRepository.FindByID(ctx, doc.Resource.ID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during resource '%s' find", doc.Resource.ID))
	}
	doc.Resource = *resource

	return s.triggerProcess(ctx, subscription, doc, kind, fn)
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
				Key:        []string{"id", "resource.id"},
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

	s.collection = "subscriptions"
	s.collectionTrigger = "subscriptionTriggers"
	s.database = s.client.Database

	return nil
}
