// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/diegobernardes/flare"
	mongodb "github.com/diegobernardes/flare/provider/mongodb"
)

type subscriptionTriggerEntity struct {
	SubscriptionID string                            `bson:"subscriptionID"`
	Document       subscriptionTriggerDocumentEntity `bson:"document"`
	Retry          subscriptionTriggerRetryEntity    `bson:"retry"`
}

type subscriptionTriggerDocumentEntity struct {
	ID        subscriptionTriggerDocumentIDEntity `bson:"id"`
	Revision  int                                 `bson:"revision"`
	UpdatedAt time.Time                           `bson:"updatedAt"`
}

type subscriptionTriggerDocumentIDEntity struct {
	Scheme string `bson:"scheme"`
	Host   string `bson:"host"`
	Path   string `bson:"path"`
}

type subscriptionTriggerRetryEntity struct {
	TTL     time.Time `bson:"ttl"`
	NextRun time.Time `bson:"nextRun"`
	Runs    int       `bson:"runs"`
}

type subscriptionTrigger struct {
	client     *mongodb.Client
	database   string
	collection string
}

func (st *subscriptionTrigger) revision(subscriptionID string, documentID url.URL) (int, error) {
	session := st.client.Session()
	defer session.Close()

	var entity subscriptionTriggerEntity
	err := session.
		DB(st.database).
		C(st.collection).
		Find(st.findQuery(subscriptionID, documentID)).
		Select(bson.M{"document.revision": 1}).
		One(&entity)
	if err != nil {
		if err == mgo.ErrNotFound {
			return 0, nil
		}
		return 0, errors.Wrap(err, "error during search")
	}

	return entity.Document.Revision, nil
}

func (st *subscriptionTrigger) delete(subscriptionID string, documentID url.URL) error {
	session := st.client.Session()
	defer session.Close()

	err := session.
		DB(st.database).
		C(st.collection).
		Remove(st.findQuery(subscriptionID, documentID))
	if err != nil {
		return errors.Wrap(err, "error during subscription triggers delete")
	}
	return nil
}

func (st *subscriptionTrigger) nextRunStatus(
	document *flare.Document, subscription subscriptionEntity,
) (bool, bool, error) {
	session := st.client.Session()
	defer session.Close()

	entity := &subscriptionTriggerEntity{}
	err := session.
		DB(st.database).
		C(st.collection).
		Find(st.findQuery(subscription.ID, document.ID)).
		One(entity)
	if err != nil {
		if err != mgo.ErrNotFound {
			return false, false, errors.Wrap(err, "error while subscription trigger search")
		}
	}
	if entity == nil {
		return false, false, nil
	}

	if entity.Retry.Runs+1 >= subscription.Delivery.Retry.Quantity {
		return false, true, nil
	}

	if entity.Retry.TTL.After(time.Now()) {
		return false, true, nil
	}

	if entity.Retry.NextRun.After(time.Now()) {
		return true, false, nil
	}

	return false, false, nil
}

func (st *subscriptionTrigger) findQuery(subscriptionID string, documentID url.URL) bson.M {
	return bson.M{
		"subscriptionID":     subscriptionID,
		"document.id.scheme": documentID.Scheme,
		"document.id.host":   documentID.Host,
		"document.id.path":   documentID.Path,
	}
}

func (st *subscriptionTrigger) exists(subscriptionID string, documentID url.URL) (bool, error) {
	session := st.client.Session()
	defer session.Close()

	count, err := session.
		DB(st.database).
		C(st.collection).
		Find(st.findQuery(subscriptionID, documentID)).
		Count()
	if err != nil {
		return false, errors.Wrap(err, "error during check if subscription trigger exists")
	}

	return count > 0, nil
}

func (st *subscriptionTrigger) incr(
	document *flare.Document, subscription *flare.Subscription,
) error {
	session := st.client.Session()
	defer session.Close()

	exists, err := st.exists(subscription.ID, document.ID)
	if err != nil {
		return err
	}

	if !exists {
		err = session.
			DB(st.database).
			C(st.collection).
			Insert(subscriptionTriggerEntity{
				SubscriptionID: subscription.ID,
				Document: subscriptionTriggerDocumentEntity{
					ID: subscriptionTriggerDocumentIDEntity{
						Scheme: document.ID.Scheme,
						Host:   document.ID.Host,
						Path:   document.ID.Path,
					},
					Revision:  (int)(document.Revision),
					UpdatedAt: time.Now(),
				},
				Retry: subscriptionTriggerRetryEntity{
					TTL:     time.Now().Add(subscription.Delivery.Retry.TTL),
					NextRun: time.Now().Add(subscription.Delivery.Retry.Interval),
					Runs:    0,
				},
			})
		if err != nil {
			return errors.Wrap(err, "error during subscription trigger insert")
		}
	}

	err = session.
		DB(st.database).
		C(st.collection).
		Update(
			st.findQuery(subscription.ID, document.ID),
			bson.M{
				"$inc": bson.M{"retry.runs": 1},
				"$set": bson.M{
					"document.updatedAt": time.Now(),
					"retry.nextRun":      time.Now().Add(subscription.Delivery.Retry.Interval),
				},
			},
		)
	if err != nil {
		return errors.Wrap(err, "error during subscription trigger update")
	}

	return nil
}

func (st *subscriptionTrigger) init() error {
	if st.client == nil {
		return errors.New("invalid client")
	}

	st.collection = "subscriptionTriggers"
	st.database = st.client.Database

	return nil
}
