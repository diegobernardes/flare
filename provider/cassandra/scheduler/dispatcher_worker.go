package scheduler

import (
	"context"
	"encoding/json"

	"github.com/gocql/gocql"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
	baseConsumer "github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/provider/cassandra"
)

type DispatcherWorker struct {
	Base *cassandra.Client
}

func (dw *DispatcherWorker) Init() error {
	if dw.Base == nil {
		return errors.New("missing Base")
	}
	return nil
}

func (dw *DispatcherWorker) FindByNodeID(
	ctx context.Context, id string,
) ([]consumer.Consumer, error) {
	iter := dw.Base.Session.Query(
		`SELECT id, created_at, payload, source, source_type FROM consumers WHERE node_id = ?`, id,
	).WithContext(ctx).Iter()

	var consumers []baseConsumer.Consumer
	for {
		consumer, err := unmarshalConsumer(iter)
		if err != nil {
			panic(err)
		}
		if consumer == nil {
			break
		}
		consumers = append(consumers, *consumer)

		if iter.WillSwitchPage() {
			break
		}
	}

	if err := iter.Close(); err != nil {
		return nil, errors.Wrap(err, "error during cassandra iter close")
	}

	return consumers, nil
}

func unmarshalConsumer(iter *gocql.Iter) (*baseConsumer.Consumer, error) {
	type payload struct {
		ID             string `json:"id"`
		RevisionField  string `json:"revisionField"`
		RevisionFormat string `json:"revisionFormat"`
	}

	var (
		c             baseConsumer.Consumer
		rawPayload    string
		rawSource     string
		rawSourceType string
	)

	if ok := iter.Scan(&c.ID, &c.CreatedAt, &rawPayload, &rawSource, &rawSourceType); !ok {
		return nil, nil
	}

	var p payload
	if err := json.Unmarshal([]byte(rawPayload), &p); err != nil {
		return nil, nil
	}

	c.Payload.ID = p.ID
	c.Payload.Revision.Field = p.RevisionField
	c.Payload.Revision.Format = p.RevisionFormat

	var err error
	switch rawSourceType {
	case "aws.sqs":
		c.Source.AWSSQS, err = unmarshalConsumerSourceAWSSQS([]byte(rawSource))
		if err != nil {
			panic(err)
		}
	case "aws.kinesis":
		c.Source.AWSKinesis, err = unmarshalConsumerSourceAWSKinesis([]byte(rawSource))
		if err != nil {
			panic(err)
		}
	}

	return &c, nil
}

func unmarshalConsumerSourceAWSSQS(data []byte) (*baseConsumer.ConsumerSourceAWSSQS, error) {
	type source struct {
		ARN         string `json:"arn"`
		Concurrency int    `json:"concurrency"`
	}
	var rawSource source
	if err := json.Unmarshal(data, &rawSource); err != nil {
		panic(err)
	}

	return &baseConsumer.ConsumerSourceAWSSQS{
		ARN:         rawSource.ARN,
		Concurrency: rawSource.Concurrency,
	}, nil
}

func unmarshalConsumerSourceAWSKinesis(data []byte) (*baseConsumer.ConsumerSourceAWSKinesis, error) {
	type source struct {
		Stream string `json:"stream"`
	}
	var rawSource source
	if err := json.Unmarshal(data, &rawSource); err != nil {
		panic(err)
	}

	return &baseConsumer.ConsumerSourceAWSKinesis{Stream: rawSource.Stream}, nil
}
