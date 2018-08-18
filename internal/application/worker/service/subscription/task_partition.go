package subscription

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// TaskPartitionConsumer handle the output of task partition process.
type TaskPartitionConsumer interface {
	Consume(ctx context.Context, docID, partitionID string) error
}

// TaskPartitionFetcher returns all the partitions a resource of a given product has.
type TaskPartitionFetcher interface {
	Fetch(ctx context.Context, docID string) ([]string, error)
}

// TaskPartition is the entrypoint of our delivery process. It receives a document id and with that
// it discover all the partitions of the document id resource and a message is generated to each
// partition.
type TaskPartition struct {
	Consumer               TaskPartitionConsumer
	MaxConsumerConcurrency uint
	Fetcher                TaskPartitionFetcher
	Enqueuer               Enqueuer
}

// Init check if the struct has everything needed to execute.
func (t TaskPartition) Init() error {
	if t.Consumer == nil {
		return errors.New("missing 'Consumer'")
	}

	if t.MaxConsumerConcurrency == 0 {
		return errors.New("invalid 'MaxConsumerConcurrency', expected to be greater then zero")
	}

	if t.Fetcher == nil {
		return errors.New("missing 'Fetcher'")
	}

	if t.Enqueuer == nil {
		return errors.New("missing 'Enqueuer'")
	}

	return nil
}

// Process receives the message from `Consume`. The message is serialized and it contains the
// document id. With it, the resources are discovered and then a new message is generated for each
// resource partition.
func (t TaskPartition) Process(ctx context.Context, payload []byte) error {
	docID, err := t.unmarshal(payload)
	if err != nil {
		return errors.Wrap(err, "unmarshal message")
	}

	partitionsID, err := t.Fetcher.Fetch(ctx, docID)
	if err != nil {
		return errors.Wrap(err, "fetch partitionsID")
	}

	if err := t.delivery(ctx, docID, partitionsID); err != nil {
		return errors.Wrap(err, "delivery")
	}

	return nil
}

// Consume receives a document id and enqueue it to be safely processed.
func (t TaskPartition) Consume(ctx context.Context, docID string) error {
	payload, err := t.marshal(docID)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	if err := t.Enqueuer.Enqueue(ctx, payload); err != nil {
		return errors.Wrap(err, "enqueue")
	}

	return nil
}

func (t TaskPartition) marshal(docID string) ([]byte, error) {
	return json.Marshal(map[string]string{"documentID": docID})
}

func (t TaskPartition) unmarshal(payload []byte) (string, error) {
	result := make(map[string]string)

	if err := json.Unmarshal(payload, &result); err != nil {
		return "", errors.Wrap(err, "unmarshal json")
	}

	docID, ok := result["documentID"]
	if !ok {
		return "", errors.New("missing document id")
	}

	return docID, nil
}

func (t TaskPartition) delivery(ctx context.Context, docID string, partitionsID []string) error {
	g, gctx := errgroup.WithContext(ctx)
	control := make(chan struct{}, t.MaxConsumerConcurrency)

	fn := func(partitionID string) func() error {
		return func() error {
			err := t.Consumer.Consume(gctx, docID, partitionID)
			<-control
			return err
		}
	}

	for _, partitionID := range partitionsID {
		control <- struct{}{}
		g.Go(fn(partitionID))
	}

	return g.Wait()
}
