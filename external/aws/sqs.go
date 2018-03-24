package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type SQS struct {
	ARN    string
	Logger log.Logger
	Base   *Client

	client   sqsiface.SQSAPI
	endpoint string
}

func (s *SQS) Init() error {
	if s.ARN == "" {
		return errors.New("invalid ARN")
	}

	if s.Logger == nil {
		return errors.New("missing Logger")
	}

	if s.Base == nil {
		return errors.New("missing Base")
	}

	// s.client = sqs.New(s.Base.session.awsSession)
	// awsClient.Config = *awsClient.Config.WithHTTPClient(client.httpClient)
	// client.sqsClient = awsClient

	return nil
}

// awsClient := sqs.New(client.session.awsSession)
// 	awsClient.Config = *awsClient.Config.WithHTTPClient(client.httpClient)
// 	client.sqsClient = awsClient

// Push content to SQS queue.
func (s *SQS) Push(ctx context.Context, content []byte) error {
	params := &sqs.SendMessageInput{
		MessageBody: aws.String(string(content)),
		QueueUrl:    aws.String(s.endpoint),
	}
	resp, err := s.client.SendMessageWithContext(ctx, params)
	if err != nil {
		return errors.Wrap(err, "error during SQS message enqueue")
	}

	level.Debug(s.Logger).Log("messageId", *resp.MessageId, "message", "Enqueued message on SQS queue")
	return nil
}

// Pull content from a SQS queue and send it to the worker.
func (s *SQS) Pull(ctx context.Context, fn func(context.Context, []byte) error) error {
	output, err := s.client.ReceiveMessageWithContext(
		ctx,
		&sqs.ReceiveMessageInput{
			AttributeNames: []*string{aws.String(sqs.QueueAttributeNameAll)},
			QueueUrl:       aws.String(s.endpoint),
		})
	if err != nil {
		return errors.Wrap(err, "error during sqs retrieve message")
	}

	var (
		deleteInput = &sqs.DeleteMessageBatchInput{
			QueueUrl: aws.String(s.endpoint),
		}

		visibilityInput = &sqs.ChangeMessageVisibilityBatchInput{
			QueueUrl: aws.String(s.endpoint),
		}

		batchId = uuid.NewV4().String()
	)

	for _, msg := range output.Messages {
		logger := log.With(s.Logger, "sqsMessageId", *msg.MessageId, "batchId", batchId)

		if err := fn(ctx, []byte(*msg.Body)); err != nil {
			level.Error(logger).Log("message", "Error during SQS message processing", "error", err)
			visibilityInput.Entries = append(
				visibilityInput.Entries, &sqs.ChangeMessageVisibilityBatchRequestEntry{
					Id:            msg.MessageId,
					ReceiptHandle: msg.ReceiptHandle,
				},
			)
			continue
		}

		level.Info(logger).Log("message", "SQS message processed")
		deleteInput.Entries = append(deleteInput.Entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		})
	}

	if err := s.deleteMessages(deleteInput, batchId); err != nil {
		return errors.Wrap(err, "error during delete processed messages")
	}

	if err := s.changeVisibilityMessages(visibilityInput, batchId); err != nil {
		return errors.Wrap(err, "error during change visibility of failed messages")
	}

	return nil
}

func (s *SQS) changeVisibilityMessages(
	req *sqs.ChangeMessageVisibilityBatchInput,
	batchId string,
) error {
	if len(req.Entries) == 0 {
		return nil
	}

	response, err := s.client.ChangeMessageVisibilityBatch(req)
	if err != nil {
		level.Error(s.Logger).Log(
			"batchId", batchId,
			"error", err,
			"message", "Error during SQS message batch change visibility",
		)
	}

	for _, err := range response.Failed {
		level.Error(s.Logger).Log(
			"batchId", batchId,
			"awsErrorSenderFault", err.SenderFault,
			"awsErrorCode", err.Code,
			"messageId", err.Id,
			"error", err.String(),
			"message", "Failed to change visibility of SQS message",
		)
	}

	for _, ok := range response.Successful {
		level.Info(s.Logger).Log(
			"batchId", batchId,
			"messageId", ok.Id,
			"message", "Changed visibility of SQS message",
		)
	}

	return nil
}

func (s *SQS) deleteMessages(req *sqs.DeleteMessageBatchInput, batchId string) error {
	if len(req.Entries) == 0 {
		return nil
	}

	response, err := s.client.DeleteMessageBatch(req)
	if err != nil {
		level.Error(s.Logger).Log(
			"batchId", batchId,
			"error", err.Error(),
			"message", "Error during SQS message batch delete",
		)
	}

	for _, err := range response.Failed {
		level.Error(s.Logger).Log(
			"batchId", batchId,
			"awsErrorSenderFault", err.SenderFault,
			"awsErrorCode", err.Code,
			"messageId", err.Id,
			"error", err.String(),
			"message", "Failed to delete SQS message",
		)
	}

	for _, ok := range response.Successful {
		level.Info(s.Logger).Log(
			"batchId", batchId,
			"messageId", ok.Id,
			"message", "Deleted SQS message",
		)
	}

	return nil
}
