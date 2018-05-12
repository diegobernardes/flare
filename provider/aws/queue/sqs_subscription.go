// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

type SQSSubscription struct {
	Base   flare.SubscriptionRepositorier
	Prefix string
	client sqsiface.SQSAPI
}

func (ss *SQSSubscription) Find(
	ctx context.Context, pagination *flare.Pagination, id string,
) ([]flare.Subscription, *flare.Pagination, error) {
	return ss.Base.Find(ctx, pagination, id)
}

func (ss *SQSSubscription) FindByID(
	ctx context.Context, resourceID, id string,
) (*flare.Subscription, error) {
	return ss.Base.FindByID(ctx, resourceID, id)
}

func (ss *SQSSubscription) FindByPartition(
	ctx context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	return ss.Base.FindByPartition(ctx, resourceID, partition)
}

func (ss *SQSSubscription) Create(ctx context.Context, subscription *flare.Subscription) error {
	queueName := fmt.Sprintf("%s-%s", ss.Prefix, subscription.ID)
	_, err := ss.client.CreateQueueWithContext(
		ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)},
	)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during queue '%s' create", queueName))
	}

	return ss.Create(ctx, subscription)
}

func (ss *SQSSubscription) Delete(ctx context.Context, resourceID, id string) error {
	queueName := fmt.Sprintf("%s-%s", ss.Prefix, id)
	_, err := ss.client.CreateQueueWithContext(
		ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)},
	)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error during queue '%s' create", queueName))
	}

	return ss.Delete(ctx, resourceID, id)
}

func (ss *SQSSubscription) Trigger(
	ctx context.Context,
	action string,
	document *flare.Document,
	subscription *flare.Subscription,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	return ss.Base.Trigger(ctx, action, document, subscription, fn)
}

/*
	flare-subscription-delivery-0d78170f-64e8-45e7-ae8f-78ac6370ce54
*/
