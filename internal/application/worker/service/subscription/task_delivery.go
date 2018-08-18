package subscription

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/internal"
)

type TaskDelivery struct {
	Enqueuer Enqueuer
}

// Init check if the struct has everything needed to execute.
func (t TaskDelivery) Init() error {
	if t.Enqueuer == nil {
		return errors.New("missing enqueuer")
	}

	return nil
}

func (t TaskDelivery) Process(ctx context.Context, payload []byte) error {
	// how the hell check if need to send a message?
	return nil
}

// do we need this?
func (t TaskDelivery) trigger(
	ctx context.Context,
	document internal.Document,
	subscription internal.Subscription,
	action string,
) error {
	return nil
}

// Consume receives a document id and a subscription id, then enqueue it to be safely processed.
func (t TaskDelivery) Consume(ctx context.Context, docID, subscriptionID string) error {
	payload, err := t.marshal(docID, subscriptionID)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	if err := t.Enqueuer.Enqueue(ctx, payload); err != nil {
		return errors.Wrap(err, "enqueue")
	}

	return nil
}

func (t TaskDelivery) marshal(docID, subscriptionID string) ([]byte, error) {
	return json.Marshal(map[string]string{
		"documentID":     docID,
		"subscriptionID": subscriptionID,
	})
}

func (t TaskDelivery) unmarshal(payload []byte) (string, string, error) {
	result := make(map[string]string)

	if err := json.Unmarshal(payload, &result); err != nil {
		return "", "", errors.Wrap(err, "unmarshal json")
	}

	docID, ok := result["documentID"]
	if !ok {
		return "", "", errors.New("missing document id")
	}

	subscriptionID, ok := result["subscriptionID"]
	if !ok {
		return "", "", errors.New("missing subscription id")
	}

	return docID, subscriptionID, nil
}
