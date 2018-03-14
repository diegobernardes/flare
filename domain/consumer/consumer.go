// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package consumer

import (
	"errors"
	"time"
)

type Consumer struct {
	ID        string
	Source    ConsumerSource
	Payload   ConsumerPayload
	NodeID    string // deveria ter isso aqui!? talvez isso seja apenas do scheduler, remover
	CreatedAt time.Time
}

// essa validacao nao tem que ser a nivel da api nao? ateh pq aqui fora ninguem vai precisar.
func (c *Consumer) Valid() error {
	if c.ID == "" {
		return errors.New("missing id")
	}

	if c.Source.AWSSQS == nil && c.Source.AWSKinesis == nil {
		return errors.New("missing source")
	}

	if c.Source.AWSSQS != nil {
		if err := c.Source.AWSSQS.Valid(); err != nil {
			return errors.New("invalid source aws.sqs")
		}
	}

	if c.Payload.ID == "" {
		return errors.New("missing payload.id")
	}

	if c.Payload.Revision.Field == "" {
		return errors.New("missing revision.field")
	}

	return nil
}

type ConsumerSource struct {
	AWSSQS     *ConsumerSourceAWSSQS
	AWSKinesis *ConsumerSourceAWSKinesis
}

type ConsumerSourceAWSSQS struct {
	ARN         string
	Concurrency int
}

func (s *ConsumerSourceAWSSQS) Valid() error {
	if s.ARN == "" {
		return errors.New("missing ARN")
	}

	if s.Concurrency <= 0 {
		return errors.New("invalid concurrency")
	}

	return nil
}

type ConsumerSourceAWSKinesis struct {
	Stream string
}

type ConsumerPayload struct {
	ID       string
	Revision ConsumerPayloadRevision
}

type ConsumerPayloadRevision struct {
	Field  string
	Format string
}
