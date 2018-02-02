// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/pkg/errors"

	baseAWS "github.com/diegobernardes/flare/provider/aws"
)

// Max size of a sqs body in bytes.
const sqsMaxMessageSize = 262144

// SQS returns a new client to interact with a SQS queue.
type SQS struct {
	name     string
	session  *baseAWS.Session
	endpoint string
	client   sqsiface.SQSAPI
}

// Push content to SQS queue.
func (s *SQS) Push(ctx context.Context, content []byte) error {
	if len(content) > sqsMaxMessageSize {
		return errors.New("document too big")
	}

	params := &sqs.SendMessageInput{
		MessageBody: aws.String(string(content)),
		QueueUrl:    aws.String(s.endpoint),
	}

	if _, err := s.client.SendMessageWithContext(ctx, params); err != nil {
		return errors.Wrap(err, "error during SQS message enqueue")
	}
	return nil
}

// Pull a message from SQS and send it to be processed.
func (s *SQS) Pull(ctx context.Context, fn func(context.Context, []byte) error) error {
	output, err := s.client.ReceiveMessageWithContext(
		ctx,
		&sqs.ReceiveMessageInput{
			AttributeNames:  []*string{aws.String(sqs.QueueAttributeNameAll)},
			QueueUrl:        aws.String(s.endpoint),
			WaitTimeSeconds: aws.Int64(20),
		},
	)
	if err != nil {
		return err
	}

	if len(output.Messages) == 0 {
		return nil
	}

	for _, msg := range output.Messages {
		if err = fn(ctx, []byte(*msg.Body)); err != nil {
			return errors.Wrap(err, "error during message process")
		}

		if _, err = s.client.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(s.endpoint),
			ReceiptHandle: msg.ReceiptHandle,
		}); err != nil {
			return errors.Wrap(err, "error during message delete")
		}
	}

	return nil
}

func (s *SQS) sqsEndpoint() (string, error) {
	result, err := s.client.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String(s.name)})
	if err != nil {
		awsErr, ok := err.(interface {
			Code() string
		})
		if !ok || (ok && awsErr.Code() != "AWS.SimpleQueueService.NonExistentQueue") {
			return "", errors.Wrap(err, fmt.Sprintf("error during check if queue '%s' exists", s.name))
		}
		return "", nil
	}

	return *result.QueueUrl, nil
}

func (s *SQS) createSQS() (string, error) {
	output, err := s.client.CreateQueue(&sqs.CreateQueueInput{QueueName: aws.String(s.name)})
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error during queue '%s' create", s.name))
	}

	for {
		queue, err := s.sqsEndpoint()
		if err != nil {
			return "", errors.Wrap(err, "error during waiting for SQS queue to be created")
		}

		if queue != "" {
			break
		}

		<-time.After(1 * time.Second)
	}

	return output.String(), nil
}

// SQSSetup is used to create the queues if not exists.
func SQSSetup(options ...func(*SQS)) error {
	s, err := initSQS(options...)
	if err != nil {
		return err
	}

	endpoint, err := s.sqsEndpoint()
	if err != nil {
		return errors.Wrap(err, "error during queue find")
	}
	if endpoint != "" {
		return nil
	}

	if _, err := s.createSQS(); err != nil {
		return errors.Wrap(err, "error during queue create")
	}
	return nil
}

// NewSQS returns a configured SQS client.
func NewSQS(options ...func(*SQS)) (*SQS, error) {
	s, err := initSQS(options...)
	if err != nil {
		return nil, err
	}

	endpoint, err := s.sqsEndpoint()
	if err != nil {
		return nil, errors.Wrap(err, "error during queue find")
	}
	if endpoint == "" {
		return nil, errors.New("queue not found")
	}
	s.endpoint = endpoint

	return s, nil
}

// SQSQueueName set the queue name.
func SQSQueueName(name string) func(*SQS) {
	return func(s *SQS) {
		s.name = name
	}
}

// SQSSession set the AWS Session.
func SQSSession(session *baseAWS.Session) func(*SQS) {
	return func(s *SQS) {
		s.session = session
	}
}

func initSQS(options ...func(*SQS)) (*SQS, error) {
	s := &SQS{}

	for _, option := range options {
		option(s)
	}

	if s.name == "" {
		return nil, errors.New("name not found")
	}

	if s.session == nil {
		return nil, errors.New("session not found")
	}
	s.client = sqs.New(s.session.Base)
	return s, nil
}
