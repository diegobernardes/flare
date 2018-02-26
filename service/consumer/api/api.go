// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/infra/pagination"
	baseConsumer "github.com/diegobernardes/flare/service/consumer"
)

type ClientRepositorier interface {
	Find(
		ctx context.Context, pagination *pagination.Pagination,
	) ([]baseConsumer.Consumer, *pagination.Pagination, error)
	FindByID(context.Context, string) (*baseConsumer.Consumer, error)
	Create(context.Context, *baseConsumer.Consumer) error
	Update(context.Context, *baseConsumer.Consumer) error
	Delete(context.Context, string) error
}

type response struct {
	Pagination *pagination.Pagination
	Consumers  []consumer
	Consumer   *consumer
}

func (r *response) MarshalJSON() ([]byte, error) {
	var result interface{}

	if r.Consumer != nil {
		result = r.Consumer
	} else {
		result = map[string]interface{}{
			"pagination": r.Pagination,
			"consumers":  r.Consumers,
		}
	}

	return json.Marshal(result)
}

type consumer baseConsumer.Consumer

func (c *consumer) MarshalJSON() ([]byte, error) {
	revision := map[string]interface{}{
		"field": c.Payload.Revision.Field,
	}

	if c.Payload.Revision.Format != "" {
		revision["format"] = c.Payload.Revision.Format
	}

	return json.Marshal(&struct {
		ID        string                 `json:"id"`
		Source    source                 `json:"source"`
		Payload   map[string]interface{} `json:"payload"`
		CreatedAt string                 `json:"createdAt"`
	}{
		ID:     c.ID,
		Source: (source)(c.Source),
		Payload: map[string]interface{}{
			"id":       c.Payload.ID,
			"revision": revision,
		},
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	})
}

type source baseConsumer.ConsumerSource

func (s *source) MarshalJSON() ([]byte, error) {
	var content map[string]interface{}

	if s.AWSKinesis != nil {
		content = map[string]interface{}{
			"type":   "aws.kinesis",
			"stream": s.AWSKinesis.Stream,
		}
	} else if s.AWSSQS != nil {
		content = map[string]interface{}{
			"type":        "aws.sqs",
			"arn":         s.AWSSQS.ARN,
			"concurrency": s.AWSSQS.Concurrency,
		}
	}

	return json.Marshal(content)
}

func transformConsumer(base *baseConsumer.Consumer) *consumer { return (*consumer)(base) }

func transformConsumers(c []baseConsumer.Consumer) []consumer {
	result := make([]consumer, len(c))
	for i := 0; i < len(c); i++ {
		result[i] = (consumer)(c[i])
	}
	return result
}

type consumerCreate struct {
	ID      string
	Source  map[string]interface{} `json:"source"`
	Payload struct {
		ID       string `json:"id"`
		Revision struct {
			Field  string `json:"field"`
			Format string `json:"format"`
		} `json:"revision"`
	} `json:"payload"`
	CreatedAt time.Time
}

func (c *consumerCreate) init() {
	c.ID = uuid.NewV4().String()
	c.CreatedAt = time.Now()
}

func (c *consumerCreate) marshal() *baseConsumer.Consumer {
	consumer := &baseConsumer.Consumer{
		ID: c.ID,
		Payload: baseConsumer.ConsumerPayload{
			ID: c.Payload.ID,
			Revision: baseConsumer.ConsumerPayloadRevision{
				Field:  c.Payload.Revision.Field,
				Format: c.Payload.Revision.Format,
			},
		},
		CreatedAt: c.CreatedAt,
	}

	switch c.Source["type"].(string) {
	case "aws.sqs":
		consumer.Source = baseConsumer.ConsumerSource{AWSSQS: c.marshalSourceAWSSQS()}
	case "aws.kinesis":
		consumer.Source = baseConsumer.ConsumerSource{AWSKinesis: c.marshalSourceAWSKinesis()}
	}

	return consumer
}

func (c *consumerCreate) marshalSourceAWSSQS() *baseConsumer.ConsumerSourceAWSSQS {
	return &baseConsumer.ConsumerSourceAWSSQS{
		ARN:         c.Source["arn"].(string),
		Concurrency: (int)(c.Source["concurrency"].(float64)),
	}
}

func (c *consumerCreate) marshalSourceAWSKinesis() *baseConsumer.ConsumerSourceAWSKinesis {
	return &baseConsumer.ConsumerSourceAWSKinesis{Stream: c.Source["stream"].(string)}
}
