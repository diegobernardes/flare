package sqs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type Client struct{}

// Push content to SQS queue.
func (c *SQSClient) Push(ctx context.Context, content []byte) error {
	message := map[string]interface{}{"Message": string(content)}
	rawContent, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "error during message json marshal")
	}

	params := &sqs.SendMessageInput{
		MessageBody: aws.String(string(rawContent)),
		QueueUrl:    aws.String(c.sqsURL),
	}
	resp, err := c.sqsClient.SendMessageWithContext(ctx, params)
	if err != nil {
		return errors.Wrap(err, "error during SQS message enqueue")
	}

	c.logger.WithValues(map[string]interface{}{
		"messageId": *resp.MessageId,
	}).Info("Enqueued message on SQS queue")
	return nil
}
