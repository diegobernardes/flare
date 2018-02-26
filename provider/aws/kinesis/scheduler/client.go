package scheduler

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"

	kinesisProvider "github.com/diegobernardes/flare/provider/aws/kinesis"
)

// Client is used to process Kinesis streams.
type Client struct {
	Base      kinesisProvider.Client
	Context   context.Context
	Processor func(string, []byte) error
}

// Start is used to process Kinesis stream processing.
func (c *Client) Start(
	ctx context.Context, stream, shardID, shardIteratorSequence string, qtd int,
) error {
	input := &kinesis.GetShardIteratorInput{
		StreamName: aws.String(stream),
		ShardId:    aws.String(shardID),
	}

	iteratorType := kinesis.ShardIteratorTypeLatest
	if shardIteratorSequence != "" {
		iteratorType = kinesis.ShardIteratorTypeAfterSequenceNumber
		input.StartingSequenceNumber = aws.String(shardIteratorSequence)
	}
	input.ShardIteratorType = aws.String(iteratorType)

	shardIterator, err := c.Base.FetchShardIterator(input)
	if err != nil {
		panic(err)
	}

	chRecord, chErr := c.Base.FetchRecords(ctx, shardIterator, qtd)
	if err := c.process(ctx, chRecord, chErr); err != nil {
		return nil
	}
	return nil
}

func (c *Client) process(
	ctx context.Context, chRecord chan []*kinesis.Record, chErr chan error,
) (err error) {
	defer func() {
		if nErr, ok := recover().(error); ok {
			err = nErr
		}
	}()

	for {
		select {
		case err := <-chErr:
			panic(err)
		case records := <-chRecord:
			for _, record := range records {
				if err := c.Processor(*record.SequenceNumber, record.Data); err != nil {
					panic(err)
				}
			}
		}
	}
}
