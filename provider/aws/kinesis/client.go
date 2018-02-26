// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kinesis

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/pkg/errors"

	providerAWS "github.com/diegobernardes/flare/provider/aws"
)

// Client implements the logic to access AWS Kinesis.
type Client struct {
	Stream  string
	Session providerAWS.Client

	base      kinesis.Kinesis
	ctx       context.Context
	ctxCancel func()
}

// FetchShards fetch all the shards a stream has.
func (c *Client) FetchShards() ([]kinesis.Shard, error) {
	var shards []kinesis.Shard
	handleStreams := func(o *kinesis.DescribeStreamOutput, hasNext bool) bool {
		for _, shard := range o.StreamDescription.Shards {
			shards = append(shards, *shard)
		}
		return hasNext
	}

	req := &kinesis.DescribeStreamInput{
		StreamName: aws.String(c.Stream),
		Limit:      aws.Int64(100),
	}
	if err := c.base.DescribeStreamPagesWithContext(c.ctx, req, handleStreams); err != nil {
		return nil, errors.Wrap(err, "error during fetch shards")
	}

	return shards, nil
}

// FetchShardIterator a iterator for a given shard.
func (c *Client) FetchShardIterator(input *kinesis.GetShardIteratorInput) (string, error) {
	iterator, err := c.base.GetShardIterator(&kinesis.GetShardIteratorInput{
		ShardId:           input.ShardId,
		ShardIteratorType: aws.String(kinesis.ShardIteratorTypeLatest),
		StreamName:        aws.String(c.Stream),
	})
	if err != nil {
		return "", errors.Wrap(err, "error during fetch kinesis shard iterator")
	}

	if iterator.ShardIterator == nil {
		return "", nil
	}
	return *iterator.ShardIterator, nil
}

// FetchRecords return a channel to consume a stream of records or any error that may occur.
func (c *Client) FetchRecords(
	ctx context.Context, iterator string, qtd int,
) (chan []*kinesis.Record, chan error) {
	output, outputErr := make(chan []*kinesis.Record), make(chan error)

	go func() {
		defer close(output)

		for {
			recordInput := &kinesis.GetRecordsInput{
				ShardIterator: aws.String(iterator),
				Limit:         aws.Int64((int64)(qtd)),
			}

			result, err := c.base.GetRecordsWithContext(ctx, recordInput)
			if err != nil {
				outputErr <- err
				break
			}

			output <- result.Records
			if result.NextShardIterator == nil {
				break
			}

			iterator = *result.NextShardIterator
		}
	}()

	return output, outputErr
}

// Init is used to initialize the client.
func (c *Client) Init() {
	c.base = *kinesis.New(c.Session.Base)
	c.ctx, c.ctxCancel = context.WithCancel(context.Background())
}
