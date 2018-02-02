package flare

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/infra/config"
	wrk "github.com/diegobernardes/flare/infra/worker"
	"github.com/diegobernardes/flare/provider/aws"
	sqsQueue "github.com/diegobernardes/flare/provider/aws/queue"
	memoryQueue "github.com/diegobernardes/flare/provider/memory/queue"
)

type queuer interface {
	Push(context.Context, []byte) error
	Pull(context.Context, func(context.Context, []byte) error) error
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

func (q *queue) pusher(name string) (wrk.Pusher, error) {
	switch q.cfg.GetString("provider.queue") {
	case providerMemory:
		return q.providerMemory(name)
	case providerAWSSQS:
		return q.providerAWSSQS(name)
	}
	return nil, nil
}

func (q *queue) puller(name string) (wrk.Puller, error) {
	switch q.cfg.GetString("provider.queue") {
	case providerMemory:
		return q.providerMemory(name)
	case providerAWSSQS:
		return q.providerAWSSQS(name)
	}
	return nil, nil
}

func (q *queue) providerMemory(name string) (queuer, error) {
	timeout, err := q.cfg.GetDuration(
		fmt.Sprintf("provider.memory.queue.%s.process-timeout", name),
	)
	if err != nil {
		return nil, err
	}

	return memoryQueue.NewClient(memoryQueue.ClientProcessTimeout(timeout)), nil
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
		return q.setupAWSSQS()
	}

	return nil
}

func (q *queue) setupAWSSQS() error {
	names := []string{"subscription.partition", "subscription.spread", "subscription.delivery"}

	logger := level.Info(q.logger)
	for _, name := range names {
		qn := q.cfg.GetString(fmt.Sprintf("provider.aws.sqs.queue.%s.queue", name))

		logger.Log("message", fmt.Sprintf("creating SQS queue for worker '%s' with name '%s'", name, qn))

		err := sqsQueue.SQSSetup(
			sqsQueue.SQSQueueName(qn),
			sqsQueue.SQSSession(q.awsSession),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
