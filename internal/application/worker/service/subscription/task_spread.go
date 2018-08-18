package subscription

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// TaskSpreadSubscriptionFetcher returns all the subscriptions of a given partition.
type TaskSpreadSubscriptionFetcher interface {
	Fetch(ctx context.Context, partitionID string) ([]string, error)
}

// TaskSpredConsumer handle the output of task spread process.
type TaskSpredConsumer interface {
	Consume(ctx context.Context, docID, subscriptionID string) error
}

// TaskSpread is responsible to discover all the subscriptions of a given partition and a message
// is generated for each of then.
type TaskSpread struct {
	Consumer               TaskSpredConsumer
	MaxConsumerConcurrency uint
	SubscriptionFetcher    TaskSpreadSubscriptionFetcher
	Enqueuer               Enqueuer
}

// Init check if the struct has everything needed to execute.
func (t TaskSpread) Init() error {
	if t.Consumer == nil {
		return errors.New("missing 'Consumer'")
	}

	if t.MaxConsumerConcurrency == 0 {
		return errors.New("invalid 'MaxConsumerConcurrency', expected to be greater then zero")
	}

	if t.SubscriptionFetcher == nil {
		return errors.New("missing 'SubscriptionFetcher'")
	}

	if t.Enqueuer == nil {
		return errors.New("missing 'Enqueuer'")
	}

	return nil
}

// Process receives the message from `Consume`. The message is serialized and it contains the
// document id and the partition id. With it, the subscriptions are discovered and the a new
// message is generated for each of then.
func (t TaskSpread) Process(ctx context.Context, payload []byte) error {
	docID, partitionID, err := t.unmarshal(payload)
	if err != nil {
		return errors.Wrap(err, "unmarshal message")
	}

	subscriptionsID, err := t.SubscriptionFetcher.Fetch(ctx, partitionID)
	if err != nil {
		return errors.Wrap(err, "fetch subscriptionsID")
	}

	if err := t.delivery(ctx, docID, subscriptionsID); err != nil {
		return errors.Wrap(err, "delivery")
	}

	return nil
}

// Consume receives a document id and a partition id, then enqueue it to be safely processed.
func (t TaskSpread) Consume(ctx context.Context, docID, partitionID string) error {
	payload, err := t.marshal(docID, partitionID)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	if err := t.Enqueuer.Enqueue(ctx, payload); err != nil {
		return errors.Wrap(err, "enqueue")
	}

	return nil
}

func (t TaskSpread) marshal(docID, partitionID string) ([]byte, error) {
	return json.Marshal(map[string]string{
		"documentID":  docID,
		"partitionID": partitionID,
	})
}

func (t TaskSpread) unmarshal(payload []byte) (string, string, error) {
	result := make(map[string]string)

	if err := json.Unmarshal(payload, &result); err != nil {
		return "", "", errors.Wrap(err, "unmarshal json")
	}

	docID, ok := result["documentID"]
	if !ok {
		return "", "", errors.New("missing document id")
	}

	partitionID, ok := result["partitionID"]
	if !ok {
		return "", "", errors.New("missing partition id")
	}

	return docID, partitionID, nil

}

func (t TaskSpread) delivery(ctx context.Context, docID string, subscriptionsID []string) error {
	g, gctx := errgroup.WithContext(ctx)
	control := make(chan struct{}, t.MaxConsumerConcurrency)

	fn := func(subscriptionID string) func() error {
		return func() error {
			err := t.Consumer.Consume(gctx, docID, subscriptionID)
			<-control
			return err
		}
	}

	for _, subscriptionID := range subscriptionsID {
		control <- struct{}{}
		g.Go(fn(subscriptionID))
	}

	return g.Wait()
}
