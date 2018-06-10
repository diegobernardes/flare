// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
	"github.com/diegobernardes/flare/provider/aws"
	sqsQueue "github.com/diegobernardes/flare/provider/aws/queue"
	memoryQueue "github.com/diegobernardes/flare/provider/memory/queue"
)

type queuer interface {
	Push(context.Context, []byte) error
	Pull(context.Context, func(context.Context, []byte) error) error
}

type queueCreator interface {
	Create(ctx context.Context, id string) error
}

type queue struct {
	cfg        *config.Client
	awsSession *aws.Session
	logger     log.Logger
}

func (q *queue) init() error {
	provider := q.cfg.GetString("provider.queue")
	switch provider {
	case providerMemory:
	case providerAWSSQS:
		var err error
		q.awsSession, err = aws.NewSession(
			aws.SessionKey(q.cfg.GetString("provider.aws.key")),
			aws.SessionRegion(q.cfg.GetString("provider.aws.region")),
			aws.SessionSecret(q.cfg.GetString("provider.aws.secret")),
		)
		if err != nil {
			return errors.Wrap(err, "error during AWS session initialization")
		}
	default:
		return fmt.Errorf("invalid provider.queue '%s' config", provider)
	}

	return nil
}

func (q *queue) fetch(name string) (queuer, error) {
	switch q.cfg.GetString("provider.queue") {
	case providerMemory:
		return memoryQueue.NewClient(), nil
	case providerAWSSQS:
		return q.providerAWSSQS(name)
	}
	return nil, nil
}

func (q *queue) providerAWSSQS(name string) (queuer, error) {
	queue, err := sqsQueue.NewSQS(
		sqsQueue.SQSQueueName(q.cfg.GetString(fmt.Sprintf("provider.aws.sqs.queue.%s.queue", name))),
		sqsQueue.SQSSession(q.awsSession),
	)
	if err != nil {
		return nil, err
	}

	return queue, nil
}

func (q *queue) setup(ctx context.Context) error {
	switch q.cfg.GetString("provider.queue") {
	case providerMemory:
	case providerAWSSQS:
		return q.setupAWSSQS(ctx)
	}

	return nil
}

// type queue interface {
// 	Push(ctx context.Context, payload []byte) error
// 	Pull(ctx context.Context, fn func(context.Context, []byte) error) error
// }

func something(queuer) error {
	return nil
}

// a ideia desse cara eh devolver alguem que consiga criar uma fila.
func (q *queue) creator() (queueCreator, error) {
	switch q.cfg.GetString("provider.queue") {
	case providerMemory:
		var x queuer

		return &memoryQueue.Client{RegisterQueue: something}, nil
	case providerAWSSQS:
	}

	return nil, nil
}

func (q *queue) setupAWSSQS(ctx context.Context) error {
	names := []string{
		"subscription.partition", "subscription.spread", "subscription.delivery", "generic",
	}

	logger := level.Info(q.logger)
	for _, name := range names {
		qn := q.cfg.GetString(fmt.Sprintf("provider.aws.sqs.queue.%s.queue", name))

		logger.Log("message", fmt.Sprintf("creating SQS queue for worker '%s' with name '%s'", name, qn))

		err := sqsQueue.SQSSetup(
			ctx,
			sqsQueue.SQSQueueName(qn),
			sqsQueue.SQSSession(q.awsSession),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
